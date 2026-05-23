package im

import (
	"context"
	"sync"
)

// MockIMChannelRepo is an in-memory implementation for testing.
type MockIMChannelRepo struct {
	mu       sync.RWMutex
	channels map[int64]*IMChannel
}

func NewMockIMChannelRepo() *MockIMChannelRepo {
	return &MockIMChannelRepo{channels: make(map[int64]*IMChannel)}
}

func (m *MockIMChannelRepo) Create(_ context.Context, c *IMChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[c.ID] = c
	return nil
}

func (m *MockIMChannelRepo) GetByID(_ context.Context, id int64) (*IMChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.channels[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *MockIMChannelRepo) Update(_ context.Context, c *IMChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[c.ID] = c
	return nil
}

func (m *MockIMChannelRepo) List(_ context.Context, tenantID int64) ([]*IMChannel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*IMChannel
	for _, c := range m.channels {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

// MockIMSessionRepo is an in-memory implementation for testing.
type MockIMSessionRepo struct {
	mu       sync.RWMutex
	sessions map[int64]*IMSession
}

func NewMockIMSessionRepo() *MockIMSessionRepo {
	return &MockIMSessionRepo{sessions: make(map[int64]*IMSession)}
}

func (m *MockIMSessionRepo) Create(_ context.Context, s *IMSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *MockIMSessionRepo) GetByID(_ context.Context, id int64) (*IMSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *MockIMSessionRepo) Update(_ context.Context, s *IMSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *MockIMSessionRepo) List(_ context.Context, tenantID int64, offset, limit int) ([]*IMSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var all []*IMSession
	for _, s := range m.sessions {
		if s.TenantID == tenantID {
			all = append(all, s)
		}
	}
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *MockIMSessionRepo) CountActiveByAgent(_ context.Context, agentUserID int64) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, s := range m.sessions {
		if s.AgentUserID != nil && *s.AgentUserID == agentUserID && s.Status == SessionStatusActive {
			count++
		}
	}
	return count, nil
}

// MockIMMessageRepo is an in-memory implementation for testing.
type MockIMMessageRepo struct {
	mu       sync.RWMutex
	messages []*IMMessage
	nextID   int64
}

func NewMockIMMessageRepo() *MockIMMessageRepo {
	return &MockIMMessageRepo{nextID: 1}
}

func (m *MockIMMessageRepo) Create(_ context.Context, msg *IMMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg.ID = m.nextID
	m.nextID++
	m.messages = append(m.messages, msg)
	return nil
}

func (m *MockIMMessageRepo) ListBySession(_ context.Context, sessionID int64, offset, limit int) ([]*IMMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*IMMessage
	for _, msg := range m.messages {
		if msg.SessionID == sessionID {
			result = append(result, msg)
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
