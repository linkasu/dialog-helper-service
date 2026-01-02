package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/auth"
	"github.com/linkasu/dialog-helper-service/internal/config"
	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/storage"
)

func main() {
	cfg, err := config.LoadStorageConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	store, err := buildStore(ctx, cfg)
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}
	defer func() {
		if err := store.Close(ctx); err != nil {
			log.Printf("store close error: %v", err)
		}
	}()

	if err := ensureAdminKey(ctx, store, cfg.AdminMasterKey); err != nil {
		log.Fatalf("admin key init failed: %v", err)
	}

	handler := storage.NewHandler(store, cfg.InternalToken)
	mux := http.NewServeMux()
	handler.Register(mux)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("storage service listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func buildStore(ctx context.Context, cfg config.StorageConfig) (storage.Store, error) {
	switch cfg.Backend {
	case "memory":
		return storage.NewMemoryStore(), nil
	case "ydb":
		provider := iam.NewDefaultProvider()
		return storage.NewYDBStore(ctx, cfg.YDBDSN, provider)
	default:
		return nil, errInvalidBackend(cfg.Backend)
	}
}

func ensureAdminKey(ctx context.Context, store storage.Store, adminMasterKey string) error {
	if adminMasterKey == "" {
		return nil
	}

	hashValue := auth.HashAPIKey(adminMasterKey)
	item, ok, err := store.GetConfig(ctx, storage.AdminMasterKeyConfigKey)
	if err != nil {
		return err
	}
	if ok {
		if item.Value != hashValue {
			log.Printf("admin key already set; ignoring ADMIN_MASTER_KEY")
		}
		return nil
	}

	return store.SetConfig(ctx, storage.AdminMasterKeyConfigKey, hashValue)
}

func errInvalidBackend(value string) error {
	return &configError{msg: "unsupported storage backend: " + value}
}

type configError struct {
	msg string
}

func (e *configError) Error() string {
	return e.msg
}
