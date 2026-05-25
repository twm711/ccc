package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/divord97/ccc/internal/domain/report"
	"github.com/divord97/ccc/pkg/wsutil"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     wsutil.CheckOrigin(),
}

// Hub manages WebSocket connections and broadcasts dashboard updates.
type Hub struct {
	dashSvc *report.DashboardService
	logger  zerolog.Logger
	mu      sync.RWMutex
	clients map[*Client]bool
}

type Client struct {
	TenantID int64
	Send     chan []byte
}

func NewHub(dashSvc *report.DashboardService, logger zerolog.Logger) *Hub {
	return &Hub{
		dashSvc: dashSvc,
		logger:  logger,
		clients: make(map[*Client]bool),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	close(c.Send)
	h.mu.Unlock()
}

// StartBroadcast sends dashboard updates every 5 seconds.
func (h *Hub) StartBroadcast(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.broadcastAll(ctx)
		}
	}
}

func (h *Hub) broadcastAll(ctx context.Context) {
	// Snapshot tenant IDs under the lock, then release before DB queries.
	h.mu.RLock()
	tenantIDs := make(map[int64]struct{})
	for c := range h.clients {
		tenantIDs[c.TenantID] = struct{}{}
	}
	h.mu.RUnlock()

	tenantData := make(map[int64][]byte, len(tenantIDs))
	for tid := range tenantIDs {
		overview, err := h.dashSvc.GetOverview(ctx, tid)
		if err != nil {
			h.logger.Error().Err(err).Int64("tenant_id", tid).Msg("dashboard broadcast failed")
			continue
		}
		tenantData[tid], _ = json.Marshal(overview)
	}

	h.mu.RLock()
	for c := range h.clients {
		data, ok := tenantData[c.TenantID]
		if !ok {
			continue
		}
		select {
		case c.Send <- data:
		default:
			h.logger.Warn().Int64("tenant_id", c.TenantID).Msg("dashboard ws: send buffer full, dropping")
		}
	}
	h.mu.RUnlock()
}

// ServeWS upgrades the HTTP connection to WebSocket and streams dashboard updates.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("dashboard ws: upgrade failed")
		return
	}

	tenantID, _ := strconv.ParseInt(r.URL.Query().Get("tenant_id"), 10, 64)
	if tenantID == 0 {
		tenantID = 1
	}

	client := &Client{TenantID: tenantID, Send: make(chan []byte, 64)}
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

	// Reader goroutine (keep connection alive, handle pongs)
	conn.SetReadLimit(512)
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
