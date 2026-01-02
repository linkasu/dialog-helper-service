package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/ydb"

	sdk "github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
)

func main() {
	dsn := os.Getenv("YDB_DSN")
	if dsn == "" {
		log.Fatal("YDB_DSN is required")
	}

	ctx := context.Background()
	provider := iam.NewDefaultProvider()
	client, err := ydb.Open(ctx, dsn, provider)
	if err != nil {
		log.Fatalf("open YDB: %v", err)
	}
	defer func() {
		if err := client.Close(ctx); err != nil {
			log.Printf("close YDB: %v", err)
		}
	}()

	queries := []string{
		`DROP TABLE config;`,
		`CREATE TABLE results (
  dialog_hash Utf8,
  user_hash Utf8,
  stt_transcript Utf8,
  answer_candidates_json Utf8,
  meta_json Utf8,
  created_at Timestamp,
  PRIMARY KEY (dialog_hash)
);`,
		`CREATE TABLE api_keys (
  api_key_hash Utf8,
  client_id Utf8,
  status Utf8,
  scopes_json Utf8,
  rate_limit_per_minute Int32,
  created_at Timestamp,
  revoked_at Timestamp?,
  PRIMARY KEY (api_key_hash)
);`,
		`CREATE TABLE config (
  config_key Utf8,
  config_value Utf8,
  updated_at Timestamp,
  PRIMARY KEY (config_key)
);`,
		`ALTER TABLE results SET TTL = Interval("PT1440M") ON created_at;`,
	}

	for _, query := range queries {
		err := executeScheme(ctx, client, query)
		if err != nil {
			log.Fatalf("schema error: %v", err)
		}
	}

	log.Printf("schema applied")
}

func executeScheme(ctx context.Context, client *sdk.Driver, query string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	return client.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		err := session.ExecuteSchemeQuery(ctx, query)
		if err == nil {
			return nil
		}
		if isAlreadyExists(err) || isExists(err) || isNotFound(err) {
			return nil
		}
		return err
	}, table.WithIdempotent())
}

func isAlreadyExists(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

func isExists(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "exists")
}

func isNotFound(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}
