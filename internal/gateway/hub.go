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
	UserID   uuid.UUID
	Conn     *websocket.Conn
	Send     chan []byte
	isClosed bool
	mu       sync.Mutex
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isClosed {
		c.isClosed = true
		// Simply close the network socket. This instantly forces writePump and readPump
		// loops to error out and exit naturally without creating a "send on closed channel" panic risk.
		_ = c.Conn.Close()
	}
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
			if current, exists := h.clients[client.UserID]; exists && current == client {
				delete(h.clients, client.UserID)
				client.Close()
			}
			h.mu.Unlock()
		}
	}
}
