package gateway

import (
	"bonfire-api/internal/cache"
	"bonfire-api/internal/message"
	"bonfire-api/internal/repository"
	"context"
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	// Real-time Messaging State
	rooms      map[uuid.UUID]map[uuid.UUID]bool // ChannelID -> map[UserID]bool
	msgService *message.Service

	cache cache.Manager
	store repository.Store
}

func NewHub(cache cache.Manager, store repository.Store, msgService *message.Service) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[uuid.UUID]map[uuid.UUID]bool),
		cache:      cache,
		store:      store,
		msgService: msgService,
	}
}

// JoinRoom registers a client to a channel topic
func (h *Hub) JoinRoom(channelID uuid.UUID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[channelID]; !ok {
		h.rooms[channelID] = make(map[uuid.UUID]bool)
	}
	h.rooms[channelID][userID] = true
}

// LeaveRoom removes a user from a specific channel's broadcast list
func (h *Hub) LeaveRoom(channelID uuid.UUID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if members, ok := h.rooms[channelID]; ok {
		delete(members, userID)
		// Cleanup empty map to save memory
		if len(members) == 0 {
			delete(h.rooms, channelID)
		}
	}
}

// BroadcastToRoom sends a message only to users subscribed to that channel
func (h *Hub) BroadcastToRoom(channelID uuid.UUID, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if members, ok := h.rooms[channelID]; ok {
		for userID := range members {
			if client, online := h.clients[userID]; online {
				select {
				case client.Send <- payload:
				default:
					go client.Close()
				}
			}
		}
	}
}

func (h *Hub) Run(ctx context.Context) {
	go h.listenRedisPresence(ctx)

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if oldClient, exists := h.clients[client.UserID]; exists {
				oldClient.Close()
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			// Remove from all rooms
			for _, users := range h.rooms {
				delete(users, client.UserID)
			}
			if current, exists := h.clients[client.UserID]; exists && current == client {
				delete(h.clients, client.UserID)
				client.Close()
			}
			h.mu.Unlock()
		}
	}
}
