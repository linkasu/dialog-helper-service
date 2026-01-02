package storage

import (
	"context"
	"time"
)

type Store interface {
	Close(ctx context.Context) error

	GetConfig(ctx context.Context, key string) (ConfigItem, bool, error)
	SetConfig(ctx context.Context, key, value string) error

	GetResult(ctx context.Context, dialogHash string) (ResultRecord, bool, error)
	PutResult(ctx context.Context, record ResultRecord) error

	GetAPIKey(ctx context.Context, apiKeyHash string) (APIKeyRecord, bool, error)
	CreateAPIKey(ctx context.Context, record APIKeyRecord) error
	RevokeAPIKey(ctx context.Context, apiKeyHash string, revokedAt time.Time) error
}
