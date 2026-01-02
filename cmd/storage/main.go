package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/auth"
	"github.com/linkasu/dialog-helper-service/internal/cloudlogging"
	"github.com/linkasu/dialog-helper-service/internal/config"
	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/storage"
)

//go:embed web
var webAssets embed.FS

func main() {
	cfg, err := config.LoadStorageConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	provider := iam.NewDefaultProvider()
	store, err := buildStore(ctx, cfg, provider)
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

	logReader, err := buildLogReader(cfg, provider)
	if err != nil {
		log.Fatalf("log reader init failed: %v", err)
	}
	if logReader != nil {
		defer func() {
			if err := logReader.Close(); err != nil {
				log.Printf("log reader close error: %v", err)
			}
		}()
	}

	handler := storage.NewHandler(store, storage.HandlerConfig{
		InternalToken:    cfg.InternalToken,
		TelegramBotToken: cfg.TelegramBotToken,
		TelegramAdminID:  cfg.TelegramAdminID,
		AdminOTPTTL:      cfg.AdminOTPTTL,
		AdminSessionTTL:  cfg.AdminSessionTTL,
		LogReader:        logReader,
	})
	mux := http.NewServeMux()
	handler.Register(mux)
	registerStatic(mux)

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

func registerStatic(mux *http.ServeMux) {
	sub, err := fs.Sub(webAssets, "web")
	if err != nil {
		log.Printf("static assets unavailable: %v", err)
		return
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
}

func buildStore(ctx context.Context, cfg config.StorageConfig, provider *iam.Provider) (storage.Store, error) {
	switch cfg.Backend {
	case "memory":
		return storage.NewMemoryStore(), nil
	case "ydb":
		if provider == nil {
			provider = iam.NewDefaultProvider()
		}
		return storage.NewYDBStore(ctx, cfg.YDBDSN, provider)
	default:
		return nil, errInvalidBackend(cfg.Backend)
	}
}

func buildLogReader(cfg config.StorageConfig, provider *iam.Provider) (*cloudlogging.Client, error) {
	if strings.TrimSpace(cfg.LogGroupID) == "" {
		return nil, nil
	}
	if provider == nil {
		provider = iam.NewDefaultProvider()
	}
	return cloudlogging.NewClient(cloudlogging.Config{
		Endpoint:         cfg.LogEndpoint,
		LogGroupID:       cfg.LogGroupID,
		ResourceTypes:    cfg.LogResourceTypes,
		ResourceIDs:      cfg.LogResourceIDs,
		StreamNames:      cfg.LogStreamNames,
		Levels:           cfg.LogLevels,
		Filter:           cfg.LogFilter,
		MaxResponseBytes: cfg.LogMaxBytes,
	}, provider)
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
