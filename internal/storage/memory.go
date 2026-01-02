package storage

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.RWMutex
	results map[string]ResultRecord
	apiKeys map[string]APIKeyRecord
	config  map[string]ConfigItem
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		results: make(map[string]ResultRecord),
		apiKeys: make(map[string]APIKeyRecord),
		config:  make(map[string]ConfigItem),
	}
}

func (m *MemoryStore) Close(_ context.Context) error {
	return nil
}

func (m *MemoryStore) GetConfig(_ context.Context, key string) (ConfigItem, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.config[key]
	return item, ok, nil
}

func (m *MemoryStore) SetConfig(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config[key] = ConfigItem{Key: key, Value: value, UpdatedAt: time.Now().UTC()}
	return nil
}

func (m *MemoryStore) GetResult(_ context.Context, dialogHash string) (ResultRecord, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, ok := m.results[dialogHash]
	return record, ok, nil
}

func (m *MemoryStore) PutResult(_ context.Context, record ResultRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[record.DialogHash] = record
	return nil
}

func (m *MemoryStore) GetAPIKey(_ context.Context, apiKeyHash string) (APIKeyRecord, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, ok := m.apiKeys[apiKeyHash]
	return record, ok, nil
}

func (m *MemoryStore) CreateAPIKey(_ context.Context, record APIKeyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiKeys[record.APIKeyHash] = record
	return nil
}

func (m *MemoryStore) RevokeAPIKey(_ context.Context, apiKeyHash string, revokedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.apiKeys[apiKeyHash]
	if !ok {
		return nil
	}
	record.Status = "revoked"
	record.RevokedAt = &revokedAt
	m.apiKeys[apiKeyHash] = record
	return nil
}
