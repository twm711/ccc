package agenthub

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Event struct {
	Type    string      `json:"type"`
	CallID  int64       `json:"call_id,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

type Client struct {
	AgentID  int64
	TenantID int64
	Send     chan []byte
}

type Hub struct {
	logger  zerolog.Logger
	mu      sync.RWMutex
	clients map[int64]map[*Client]bool // agentID -> clients
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		logger:  logger,
		clients: make(map[int64]map[*Client]bool),
	}
}

func (h *Hub) StartBroadcast(ctx context.Context) {
	<-ctx.Done()
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.AgentID] == nil {
		h.clients[c.AgentID] = make(map[*Client]bool)
	}
	h.clients[c.AgentID][c] = true
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[c.AgentID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.AgentID)
		}
	}
	close(c.Send)
}

// NotifyAgent implements lifecycle.AgentNotifier.
func (h *Hub) NotifyAgent(agentID int64, eventType string, callID int64, payload interface{}) {
	h.SendToAgent(agentID, Event{Type: eventType, CallID: callID, Payload: payload})
}

func (h *Hub) SendToAgent(agentID int64, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[agentID] {
		select {
		case c.Send <- data:
		default:
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("agent-events ws: upgrade failed")
		return
	}

	agentID, _ := strconv.ParseInt(r.URL.Query().Get("agent_id"), 10, 64)
	tenantID, _ := strconv.ParseInt(r.URL.Query().Get("tenant_id"), 10, 64)

	client := &Client{AgentID: agentID, TenantID: tenantID, Send: make(chan []byte, 256)}
	h.Register(client)

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

	conn.SetReadLimit(4096)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
