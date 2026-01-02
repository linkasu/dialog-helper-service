package ydb

import (
	"context"
	"fmt"

	sdk "github.com/ydb-platform/ydb-go-sdk/v3"
)

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type iamCredentials struct {
	provider TokenProvider
}

func (c iamCredentials) Token(ctx context.Context) (string, error) {
	return c.provider.Token(ctx)
}

func Open(ctx context.Context, dsn string, provider TokenProvider) (*sdk.Driver, error) {
	if dsn == "" {
		return nil, fmt.Errorf("YDB DSN is required")
	}

	opts := []sdk.Option{}
	if provider != nil {
		opts = append(opts, sdk.WithCredentials(iamCredentials{provider: provider}))
	}

	db, err := sdk.Open(ctx, dsn, opts...)
	if err != nil {
		return nil, err
	}

	return db, nil
}
