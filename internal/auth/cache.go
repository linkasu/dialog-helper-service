package auth

import (
	"sync"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/models"
)

type KeyCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]cacheEntry
}

type cacheEntry struct {
	info      models.APIKeyInfo
	expiresAt time.Time
}

func NewKeyCache(ttl time.Duration) *KeyCache {
	return &KeyCache{
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
	}
}

func (c *KeyCache) Get(hash string) (models.APIKeyInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[hash]
	if !ok {
		return models.APIKeyInfo{}, false
	}
	if time.Now().UTC().After(entry.expiresAt) {
		delete(c.entries, hash)
		return models.APIKeyInfo{}, false
	}
	return entry.info, true
}

func (c *KeyCache) Set(hash string, info models.APIKeyInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[hash] = cacheEntry{info: info, expiresAt: time.Now().UTC().Add(c.ttl)}
}
