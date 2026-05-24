package imhub

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

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
	conn      *websocket.Conn
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
		if _, exists := clients[c]; exists {
			delete(clients, c)
			close(c.Send)
		}
		if len(clients) == 0 {
			delete(h.clients, c.SessionID)
		}
	}
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

// ServeWS upgrades HTTP to WebSocket for IM messaging.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("im ws upgrade failed")
		return
	}

	clientID := r.URL.Query().Get("client_id")
	sessionID, _ := strconv.ParseInt(r.URL.Query().Get("session_id"), 10, 64)

	c := &Client{
		ID:        clientID,
		SessionID: sessionID,
		Send:      make(chan []byte, 64),
		conn:      conn,
	}
	h.Register(c)

	go h.writePump(c)
	go h.readPump(c)
}

func (h *Hub) writePump(c *Client) {
	defer c.conn.Close()
	for msg := range c.Send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (h *Hub) readPump(c *Client) {
	defer func() {
		h.Unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}
