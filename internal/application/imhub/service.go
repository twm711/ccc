package imhub

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/rs/zerolog"
)

// IMEvent represents a message event sent over WebSocket.
type IMEvent struct {
	Type      string      `json:"type"`
	SessionID int64       `json:"session_id"`
	Payload   interface{} `json:"payload"`
}

// Client represents a connected WebSocket client (agent or visitor).
type Client struct {
	ID        string
	SessionID int64
	Send      chan []byte
}

// Hub manages WebSocket connections for IM real-time messaging.
type Hub struct {
	logger  zerolog.Logger
	mu      sync.RWMutex
	clients map[int64]map[*Client]bool // sessionID -> clients
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		logger:  logger,
		clients: make(map[int64]map[*Client]bool),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.SessionID] == nil {
		h.clients[c.SessionID] = make(map[*Client]bool)
	}
	h.clients[c.SessionID][c] = true
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[c.SessionID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.SessionID)
		}
	}
	close(c.Send)
}

// Broadcast sends an event to all clients in a session.
func (h *Hub) Broadcast(sessionID int64, event IMEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error().Err(err).Msg("im hub: marshal event failed")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := h.clients[sessionID]
	for c := range clients {
		select {
		case c.Send <- data:
		default:
			h.logger.Warn().Str("client", c.ID).Msg("im hub: client send buffer full, dropping")
		}
	}
}

// ServeWS is a placeholder for WebSocket upgrade handling.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	// In production, use gorilla/websocket or nhooyr.io/websocket for upgrade.
	// This is a structural placeholder — actual WebSocket upgrade is deferred
	// to frontend integration phase.
	http.Error(w, "websocket endpoint: use a WebSocket client", http.StatusUpgradeRequired)
}
