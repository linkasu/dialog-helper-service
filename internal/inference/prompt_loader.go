package inference

import (
	"context"
	"sync"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/storage"
)

type ConfigClient interface {
	GetConfig(ctx context.Context, key string) (storage.ConfigItem, bool, error)
}

type PromptInfo struct {
	Prompt  string
	Version string
}

type PromptLoader struct {
	client   ConfigClient
	ttl      time.Duration
	fallback string
	cached   PromptInfo
	expires  time.Time
	mu       sync.Mutex
}

func NewPromptLoader(client ConfigClient, ttl time.Duration, fallback string) *PromptLoader {
	return &PromptLoader{client: client, ttl: ttl, fallback: fallback}
}

func (l *PromptLoader) Get(ctx context.Context) (PromptInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	if l.cached.Prompt != "" && now.Before(l.expires) {
		return l.cached, nil
	}

	promptItem, ok, err := l.client.GetConfig(ctx, storage.SystemPromptConfigKey)
	if err != nil {
		return PromptInfo{}, err
	}
	versionItem, _, err := l.client.GetConfig(ctx, storage.SystemPromptVersionConfigKey)
	if err != nil {
		return PromptInfo{}, err
	}

	prompt := ""
	if ok {
		prompt = promptItem.Value
	}
	if prompt == "" {
		prompt = l.fallback
	}

	info := PromptInfo{Prompt: prompt, Version: versionItem.Value}
	l.cached = info
	l.expires = now.Add(l.ttl)

	return info, nil
}
