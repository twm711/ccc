package call

import (
	"context"
	"sync"
)

type MockCallRepo struct {
	mu    sync.RWMutex
	calls map[int64]*Call
}

func NewMockCallRepo() *MockCallRepo {
	return &MockCallRepo{calls: make(map[int64]*Call)}
}

func (r *MockCallRepo) Create(_ context.Context, c *Call) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls[c.ID] = c
	return nil
}

func (r *MockCallRepo) GetByID(_ context.Context, id int64) (*Call, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls[id], nil
}

func (r *MockCallRepo) Update(_ context.Context, c *Call) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls[c.ID] = c
	return nil
}

func (r *MockCallRepo) List(_ context.Context, tenantID int64, offset, limit int) ([]*Call, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []*Call
	for _, c := range r.calls {
		if c.TenantID == tenantID {
			filtered = append(filtered, c)
		}
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

type MockCallEventRepo struct {
	mu     sync.RWMutex
	events []*CallEvent
}

func NewMockCallEventRepo() *MockCallEventRepo {
	return &MockCallEventRepo{}
}

func (r *MockCallEventRepo) Create(_ context.Context, e *CallEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

func (r *MockCallEventRepo) ListByCallID(_ context.Context, callID int64) ([]*CallEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*CallEvent
	for _, e := range r.events {
		if e.CallID == callID {
			result = append(result, e)
		}
	}
	return result, nil
}

type MockIVRTrackingRepo struct {
	mu      sync.RWMutex
	entries []*IVRTracking
}

func NewMockIVRTrackingRepo() *MockIVRTrackingRepo {
	return &MockIVRTrackingRepo{}
}

func (r *MockIVRTrackingRepo) Create(_ context.Context, t *IVRTracking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, t)
	return nil
}

func (r *MockIVRTrackingRepo) ListByCallID(_ context.Context, callID int64) ([]*IVRTracking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*IVRTracking
	for _, t := range r.entries {
		if t.CallID == callID {
			result = append(result, t)
		}
	}
	return result, nil
}
