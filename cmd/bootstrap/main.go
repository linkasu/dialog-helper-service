package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/auth"
	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/storage"
)

func main() {
	var (
		systemPrompt     string
		systemPromptFile string
	)
	flag.StringVar(&systemPrompt, "system-prompt", "", "system prompt text")
	flag.StringVar(&systemPromptFile, "system-prompt-file", "", "path to file with system prompt")
	flag.Parse()

	if systemPrompt == "" && systemPromptFile != "" {
		data, err := os.ReadFile(systemPromptFile)
		if err != nil {
			log.Fatalf("read system prompt file: %v", err)
		}
		systemPrompt = string(data)
	}

	dsn := os.Getenv("YDB_DSN")
	if dsn == "" {
		log.Fatal("YDB_DSN is required")
	}

	ctx := context.Background()
	provider := iam.NewDefaultProvider()
	store, err := storage.NewYDBStore(ctx, dsn, provider)
	if err != nil {
		log.Fatalf("open YDB: %v", err)
	}
	defer func() {
		if err := store.Close(ctx); err != nil {
			log.Printf("close YDB: %v", err)
		}
	}()

	adminKey, adminHash := ensureAdminKey(ctx, store)
	if adminKey != "" {
		fmt.Printf("ADMIN_MASTER_KEY=%s\n", adminKey)
	}

	if systemPrompt != "" {
		if err := store.SetConfig(ctx, storage.SystemPromptConfigKey, systemPrompt); err != nil {
			log.Fatalf("set system prompt: %v", err)
		}
		version := time.Now().UTC().Format("20060102150405")
		if err := store.SetConfig(ctx, storage.SystemPromptVersionConfigKey, version); err != nil {
			log.Fatalf("set system prompt version: %v", err)
		}
		log.Printf("system prompt version set to %s", version)
	}

	if adminHash == "" {
		log.Printf("admin master key already exists")
	}
}

func ensureAdminKey(ctx context.Context, store storage.Store) (string, string) {
	item, ok, err := store.GetConfig(ctx, storage.AdminMasterKeyConfigKey)
	if err != nil {
		log.Fatalf("get admin key config: %v", err)
	}
	if ok && item.Value != "" {
		return "", item.Value
	}

	plain, hashValue, err := auth.GenerateAPIKey()
	if err != nil {
		log.Fatalf("generate admin key: %v", err)
	}
	if err := store.SetConfig(ctx, storage.AdminMasterKeyConfigKey, hashValue); err != nil {
		log.Fatalf("set admin key hash: %v", err)
	}

	return plain, hashValue
}
