package gateway

import (
	"context"
	"sync"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	cache cache.Manager
	store repository.Store
}

func NewHub(cache cache.Manager, store repository.Store) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		cache:      cache,
		store:      store,
	}
}

func (h *Hub) Run(ctx context.Context) {
	// Start the Redis Pub/Sub listener background worker
	go h.listenRedisPresence(ctx)

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// If user has an old session open, close it cleanly first
			if oldClient, exists := h.clients[client.UserID]; exists {
				close(oldClient.Send)
				oldClient.Conn.Close()
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, exists := h.clients[client.UserID]; exists {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
		}
	}
}
