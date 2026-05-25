package transcripthub

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/divord97/ccc/pkg/wsutil"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     wsutil.CheckOrigin(),
}

type TranscriptEvent struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	Role      string `json:"role,omitempty"`
	Text      string `json:"text,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Sentiment string `json:"sentiment,omitempty"`
}

type Client struct {
	CallID int64
	Send   chan []byte
}

type Hub struct {
	logger  zerolog.Logger
	mu      sync.RWMutex
	clients map[int64]map[*Client]bool // callID -> clients
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
	if h.clients[c.CallID] == nil {
		h.clients[c.CallID] = make(map[*Client]bool)
	}
	h.clients[c.CallID][c] = true
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[c.CallID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.CallID)
		}
	}
	close(c.Send)
}

func (h *Hub) Broadcast(callID int64, event TranscriptEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[callID] {
		select {
		case c.Send <- data:
		default:
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("transcript ws: upgrade failed")
		return
	}

	callID, _ := strconv.ParseInt(r.URL.Query().Get("call_id"), 10, 64)

	client := &Client{CallID: callID, Send: make(chan []byte, 256)}
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
