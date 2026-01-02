package storage

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/auth"
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

func (m *MemoryStore) ListResults(_ context.Context, opts ResultListOptions) ([]ResultRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]ResultRecord, 0, len(m.results))
	for _, record := range m.results {
		if !opts.Since.IsZero() && record.CreatedAt.Before(opts.Since) {
			continue
		}
		results = append(results, record)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	limit := opts.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	return append([]ResultRecord(nil), results[:limit]...), nil
}

func (m *MemoryStore) CountResults(_ context.Context, since time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, record := range m.results {
		if !since.IsZero() && record.CreatedAt.Before(since) {
			continue
		}
		count++
	}
	return count, nil
}

func (m *MemoryStore) CountActiveUsers(_ context.Context, since time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	seen := make(map[string]struct{})
	for _, record := range m.results {
		if !since.IsZero() && record.CreatedAt.Before(since) {
			continue
		}
		if record.UserHash == "" {
			continue
		}
		seen[record.UserHash] = struct{}{}
	}
	return len(seen), nil
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
	record.Status = auth.StatusRevoked
	record.RevokedAt = &revokedAt
	m.apiKeys[apiKeyHash] = record
	return nil
}

func (m *MemoryStore) ListAPIKeys(_ context.Context, opts APIKeyListOptions) ([]APIKeyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]APIKeyRecord, 0, len(m.apiKeys))
	for _, record := range m.apiKeys {
		if !opts.IncludeRevoked && record.Status == auth.StatusRevoked {
			continue
		}
		keys = append(keys, record)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].CreatedAt.After(keys[j].CreatedAt)
	})

	limit := opts.Limit
	if limit <= 0 || limit > len(keys) {
		limit = len(keys)
	}
	return append([]APIKeyRecord(nil), keys[:limit]...), nil
}
