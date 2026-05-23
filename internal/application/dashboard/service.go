package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/divord97/ccc/internal/domain/report"
	"github.com/rs/zerolog"
)

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
	h.mu.RLock()
	defer h.mu.RUnlock()

	tenantData := make(map[int64][]byte)

	for c := range h.clients {
		data, ok := tenantData[c.TenantID]
		if !ok {
			overview, err := h.dashSvc.GetOverview(ctx, c.TenantID)
			if err != nil {
				h.logger.Error().Err(err).Int64("tenant_id", c.TenantID).Msg("dashboard broadcast failed")
				continue
			}
			data, _ = json.Marshal(overview)
			tenantData[c.TenantID] = data
		}

		select {
		case c.Send <- data:
		default:
			// Client send buffer full, skip
		}
	}
}

// ServeWS upgrades the HTTP connection to WebSocket (placeholder — actual upgrade in handler).
func (h *Hub) ServeWS(_ http.ResponseWriter, _ *http.Request) {
	// WebSocket upgrade is handled in the HTTP handler layer.
}
