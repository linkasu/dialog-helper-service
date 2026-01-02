package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/config"
	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/inference"
	"github.com/linkasu/dialog-helper-service/internal/providers"
	"github.com/linkasu/dialog-helper-service/internal/storageclient"
)

func main() {
	cfg, err := config.LoadInferenceConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	storageClient, err := storageclient.New(cfg.StorageURL, cfg.InternalToken)
	if err != nil {
		log.Fatalf("storage client error: %v", err)
	}

	promptLoader := inference.NewPromptLoader(storageClient, cfg.ConfigCacheTTL, cfg.SystemPromptFallback)
	provider := iam.NewDefaultProvider()

	sttClient := buildSTT(cfg, provider)
	llmClient := buildLLM(cfg, provider)

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

func buildSTT(cfg config.InferenceConfig, provider *iam.Provider) providers.STTClient {
	switch strings.ToLower(cfg.STTProvider) {
	case "mock":
		return providers.NewMockSTT(os.Getenv("MOCK_STT_TRANSCRIPT"))
	default:
		return providers.NewYandexSTT(cfg.FolderID, cfg.STTModel, provider, cfg.STTPollInterval, cfg.STTPollTimeout)
	}
}

func buildLLM(cfg config.InferenceConfig, provider *iam.Provider) providers.LLMClient {
	switch strings.ToLower(cfg.LLMProvider) {
	case "mock":
		texts := parseMockTexts(os.Getenv("MOCK_LLM_TEXTS"))
		return providers.NewMockLLM(texts)
	default:
		return providers.NewYandexLLM(cfg.FolderID, provider)
	}
}

func parseMockTexts(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
