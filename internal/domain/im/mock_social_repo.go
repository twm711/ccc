package im

import (
	"context"
	"sync"
)

// MockSocialChannelConfigRepo is an in-memory mock for testing.
type MockSocialChannelConfigRepo struct {
	mu    sync.RWMutex
	items map[int64]*SocialChannelConfig
}

func NewMockSocialChannelConfigRepo() *MockSocialChannelConfigRepo {
	return &MockSocialChannelConfigRepo{items: make(map[int64]*SocialChannelConfig)}
}

func (m *MockSocialChannelConfigRepo) Create(_ context.Context, c *SocialChannelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[c.ID] = c
	return nil
}

func (m *MockSocialChannelConfigRepo) GetByChannelID(_ context.Context, channelID int64) (*SocialChannelConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.items {
		if c.ChannelID == channelID {
			return c, nil
		}
	}
	return nil, nil
}

func (m *MockSocialChannelConfigRepo) GetByPlatformAndAppID(_ context.Context, platform SocialPlatform, appID string) (*SocialChannelConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.items {
		if c.Platform == platform && c.AppID == appID {
			return c, nil
		}
	}
	return nil, nil
}

func (m *MockSocialChannelConfigRepo) Update(_ context.Context, c *SocialChannelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[c.ID] = c
	return nil
}

func (m *MockSocialChannelConfigRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}
