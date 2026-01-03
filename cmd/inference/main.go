package main

import (
	"log"
	"net/http"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/config"
	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/inference"
	"github.com/linkasu/dialog-helper-service/internal/storageclient"
)

func main() {
	cfg, err := config.LoadInferenceConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	provider := iam.NewDefaultProvider()

	storageClient, err := storageclient.New(cfg.StorageURL, cfg.InternalToken, provider)
	if err != nil {
		log.Fatalf("storage client error: %v", err)
	}

	promptLoader := inference.NewPromptLoader(storageClient, cfg.ConfigCacheTTL, cfg.SystemPromptFallback)

	sttClient := inference.BuildSTT(cfg, provider)
	llmClient := inference.BuildLLM(cfg, provider)

	handler := inference.NewHandler(cfg, storageClient, sttClient, llmClient, promptLoader)
	mux := http.NewServeMux()
	handler.Register(mux)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("inference service listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
