package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow checks if a request is allowed under token bucket rate limiting.
// Returns true if allowed, false if rate limited.
func (rl *RateLimiter) Allow(ctx context.Context, tenantID int64, ratePerSec int) (bool, error) {
	key := fmt.Sprintf("ratelimit:tenant:%d", tenantID)
	now := time.Now().Unix()

	script := redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local window = 1

		local count = redis.call('GET', key)
		if count and tonumber(count) >= rate then
			return 0
		end

		redis.call('INCR', key)
		if not count then
			redis.call('EXPIRE', key, window)
		end
		return 1
	`)

	result, err := script.Run(ctx, rl.client, []string{key}, ratePerSec, now).Int()
	if err != nil {
		return true, err
	}
	return result == 1, nil
}

func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
