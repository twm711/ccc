package agenthub

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/divord97/ccc/pkg/metrics"
	"github.com/divord97/ccc/pkg/wsutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     wsutil.CheckOrigin(),
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
	logger    zerolog.Logger
	jwtSecret string
	mu        sync.RWMutex
	clients   map[int64]map[*Client]bool // agentID -> clients
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		logger:  logger,
		clients: make(map[int64]map[*Client]bool),
	}
}

// SetJWTSecret enables JWT-based authentication for WebSocket connections.
func (h *Hub) SetJWTSecret(secret string) {
	h.jwtSecret = secret
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
	metrics.WSActiveConnections.WithLabelValues("agent").Inc()
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
	metrics.WSActiveConnections.WithLabelValues("agent").Dec()
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
			h.logger.Warn().Int64("agent_id", agentID).Msg("ws: send buffer full, dropping message")
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	var agentID, tenantID int64

	// Prefer JWT-based auth from Authorization header or Sec-WebSocket-Protocol.
	if h.jwtSecret != "" {
		tokenStr := ""
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		} else if proto := r.Header.Get("Sec-WebSocket-Protocol"); proto != "" {
			// Browsers can't set Authorization on WS; use subprotocol as fallback.
			tokenStr = proto
		}
		if tokenStr != "" {
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return []byte(h.jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if v, ok := claims["agent_id"].(float64); ok {
					agentID = int64(v)
				}
				if v, ok := claims["tenant_id"].(float64); ok {
					tenantID = int64(v)
				}
			}
		}
	}

	// Fallback to query params for backward compatibility.
	if agentID == 0 {
		agentID, _ = strconv.ParseInt(r.URL.Query().Get("agent_id"), 10, 64)
		tenantID, _ = strconv.ParseInt(r.URL.Query().Get("tenant_id"), 10, 64)
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("agent-events ws: upgrade failed")
		return
	}

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
