package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ConcurrencyGuard tracks and limits concurrent calls per tenant using Redis.
type ConcurrencyGuard struct {
	client *redis.Client
}

func NewConcurrencyGuard(client *redis.Client) *ConcurrencyGuard {
	return &ConcurrencyGuard{client: client}
}

func concurrencyKey(tenantID int64) string {
	return fmt.Sprintf("ccc:concurrent_calls:%d", tenantID)
}

// Acquire increments the active call counter for a tenant. Returns true if
// the counter is within the allowed limit, false if the tenant already has
// maxConcurrent active calls. When false the counter is decremented back.
func (g *ConcurrencyGuard) Acquire(ctx context.Context, tenantID int64, maxConcurrent int) (bool, error) {
	key := concurrencyKey(tenantID)
	val, err := g.client.Incr(ctx, key).Result()
	if err != nil {
		return true, err // fail open
	}
	if maxConcurrent > 0 && val > int64(maxConcurrent) {
		g.client.Decr(ctx, key)
		return false, nil
	}
	return true, nil
}

// Release decrements the active call counter for a tenant.
func (g *ConcurrencyGuard) Release(ctx context.Context, tenantID int64) {
	key := concurrencyKey(tenantID)
	g.client.Decr(ctx, key)
}
