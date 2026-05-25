package ticket

import (
	"context"
	"sync"
)

type MockCategoryRepo struct {
	mu   sync.RWMutex
	data map[int64]*TicketCategory
}

func NewMockCategoryRepo() *MockCategoryRepo {
	return &MockCategoryRepo{data: make(map[int64]*TicketCategory)}
}

func (r *MockCategoryRepo) Create(_ context.Context, c *TicketCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[c.ID] = c
	return nil
}

func (r *MockCategoryRepo) List(_ context.Context, tenantID int64) ([]*TicketCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*TicketCategory
	for _, c := range r.data {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (r *MockCategoryRepo) GetByID(_ context.Context, id int64) (*TicketCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.data[id]; ok {
		return c, nil
	}
	return nil, nil
}

type MockTemplateRepo struct {
	mu   sync.RWMutex
	data map[int64]*TicketTemplate
}

func NewMockTemplateRepo() *MockTemplateRepo {
	return &MockTemplateRepo{data: make(map[int64]*TicketTemplate)}
}

func (r *MockTemplateRepo) Create(_ context.Context, t *TicketTemplate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.ID] = t
	return nil
}

func (r *MockTemplateRepo) GetByID(_ context.Context, id int64) (*TicketTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.data[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (r *MockTemplateRepo) Update(_ context.Context, t *TicketTemplate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.ID] = t
	return nil
}

func (r *MockTemplateRepo) List(_ context.Context, tenantID int64) ([]*TicketTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*TicketTemplate
	for _, t := range r.data {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

type MockTicketRepo struct {
	mu   sync.RWMutex
	data map[int64]*Ticket
}

func NewMockTicketRepo() *MockTicketRepo {
	return &MockTicketRepo{data: make(map[int64]*Ticket)}
}

func (r *MockTicketRepo) Create(_ context.Context, t *Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.ID] = t
	return nil
}

func (r *MockTicketRepo) GetByID(_ context.Context, id int64) (*Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.data[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (r *MockTicketRepo) Update(_ context.Context, t *Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.ID] = t
	return nil
}

func (r *MockTicketRepo) List(_ context.Context, tenantID int64, offset, limit int) ([]*Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Ticket
	for _, t := range r.data {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	if offset >= len(result) {
		return nil, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (r *MockTicketRepo) ListByCallID(_ context.Context, callID int64) ([]*Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Ticket
	for _, t := range r.data {
		if t.CallID != nil && *t.CallID == callID {
			result = append(result, t)
		}
	}
	return result, nil
}

type MockCommentRepo struct {
	mu   sync.RWMutex
	data map[int64]*TicketComment
	seq  int64
}

func NewMockCommentRepo() *MockCommentRepo {
	return &MockCommentRepo{data: make(map[int64]*TicketComment)}
}

func (r *MockCommentRepo) Create(_ context.Context, c *TicketComment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	if c.ID == 0 {
		c.ID = r.seq
	}
	r.data[c.ID] = c
	return nil
}

func (r *MockCommentRepo) ListByTicket(_ context.Context, ticketID int64) ([]*TicketComment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*TicketComment
	for _, c := range r.data {
		if c.TicketID == ticketID {
			result = append(result, c)
		}
	}
	return result, nil
}
