package imhub

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/divord97/ccc/pkg/wsutil"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var imUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     wsutil.CheckOrigin(),
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
}

// Hub manages WebSocket connections for IM real-time messaging.
type Hub struct {
	imSvc   *im.IMService
	logger  zerolog.Logger
	mu      sync.RWMutex
	clients map[int64]map[*Client]bool // sessionID -> clients
}

func NewHub(imSvc *im.IMService, logger zerolog.Logger) *Hub {
	return &Hub{
		imSvc:   imSvc,
		logger:  logger,
		clients: make(map[int64]map[*Client]bool),
	}
}

// StartBroadcast keeps the hub alive; IM events are push-driven via Broadcast().
func (h *Hub) StartBroadcast(ctx context.Context) {
	<-ctx.Done()
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

// BroadcastEvent fans an IM event out to every client subscribed to the
// session. It satisfies handler.IMBroadcaster, keeping the wire format
// (`{"type", "session_id", "payload"}`) identical to what arrives over WS.
func (h *Hub) BroadcastEvent(sessionID int64, eventType string, payload interface{}) {
	h.Broadcast(sessionID, IMEvent{Type: eventType, SessionID: sessionID, Payload: payload})
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

// ServeWS upgrades the HTTP connection to WebSocket for real-time IM messaging.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := imUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("im ws: upgrade failed")
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = r.RemoteAddr
	}
	sessionID := int64(0)
	if v := r.URL.Query().Get("session_id"); v != "" {
		json.Unmarshal([]byte(v), &sessionID)
	}

	client := &Client{ID: clientID, SessionID: sessionID, Send: make(chan []byte, 256)}
	h.Register(client)

	// Writer goroutine (sends messages + ping)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
			conn.Close()
			h.Unregister(client)
		}()
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, nil)
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Reader goroutine (receive messages from client)
	conn.SetReadLimit(8192)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var event IMEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			continue
		}
		if event.SessionID == 0 {
			event.SessionID = client.SessionID
		}
		h.Broadcast(event.SessionID, event)
	}
}
