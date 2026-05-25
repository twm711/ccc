package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IVRContextStore persists IVR session variables (caller intent, DTMF choices,
// answered prompts) keyed by callID so the agent screen pop can show the
// caller's IVR journey when the call hands off to a human.
type IVRContextStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewIVRContextStore returns a Redis-backed IVR context store with the given TTL.
// Default TTL of 1h is sufficient for the longest queue waits plus an agent's ACW.
func NewIVRContextStore(rdb *redis.Client) *IVRContextStore {
	return &IVRContextStore{rdb: rdb, ttl: time.Hour}
}

func ivrContextKey(callID int64) string {
	return fmt.Sprintf("ccc:ivr:context:%d", callID)
}

// Save serializes vars and writes them under the call's context key.
func (s *IVRContextStore) Save(ctx context.Context, callID int64, vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}
	data, err := json.Marshal(vars)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, ivrContextKey(callID), data, s.ttl).Err()
}

// Load returns the previously-saved IVR variables for a call, or nil if absent.
func (s *IVRContextStore) Load(ctx context.Context, callID int64) (map[string]string, error) {
	data, err := s.rdb.Get(ctx, ivrContextKey(callID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var vars map[string]string
	if err := json.Unmarshal(data, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}
