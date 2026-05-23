package routing

import (
	"context"
	"sync"
)

type MockFlowRepo struct {
	mu    sync.RWMutex
	flows map[int64]*IVRFlow
}

func NewMockFlowRepo() *MockFlowRepo {
	return &MockFlowRepo{flows: make(map[int64]*IVRFlow)}
}

func (r *MockFlowRepo) Create(_ context.Context, f *IVRFlow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flows[f.ID] = f
	return nil
}

func (r *MockFlowRepo) GetByID(_ context.Context, id int64) (*IVRFlow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.flows[id], nil
}

func (r *MockFlowRepo) GetByCode(_ context.Context, tenantID int64, code string, version int) (*IVRFlow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, f := range r.flows {
		if f.TenantID == tenantID && f.Code == code {
			return f, nil
		}
	}
	return nil, nil
}

func (r *MockFlowRepo) Update(_ context.Context, f *IVRFlow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flows[f.ID] = f
	return nil
}

func (r *MockFlowRepo) List(_ context.Context, tenantID int64, offset, limit int) ([]*IVRFlow, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []*IVRFlow
	for _, f := range r.flows {
		if f.TenantID == tenantID {
			filtered = append(filtered, f)
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

func (r *MockFlowRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.flows, id)
	return nil
}

type MockVersionRepo struct {
	mu       sync.RWMutex
	versions []*IVRFlowVersion
	nextID   int64
}

func NewMockVersionRepo() *MockVersionRepo {
	return &MockVersionRepo{}
}

func (r *MockVersionRepo) Create(_ context.Context, v *IVRFlowVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	v.ID = r.nextID
	r.versions = append(r.versions, v)
	return nil
}

func (r *MockVersionRepo) GetByFlowAndVersion(_ context.Context, flowID int64, version int) (*IVRFlowVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.versions {
		if v.IVRFlowID == flowID && v.Version == version {
			return v, nil
		}
	}
	return nil, nil
}

func (r *MockVersionRepo) ListByFlow(_ context.Context, flowID int64) ([]*IVRFlowVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*IVRFlowVersion
	for _, v := range r.versions {
		if v.IVRFlowID == flowID {
			result = append(result, v)
		}
	}
	return result, nil
}
