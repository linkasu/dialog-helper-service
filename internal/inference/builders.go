package inference

import (
	"os"
	"strings"

	"github.com/linkasu/dialog-helper-service/internal/config"
	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/providers"
)

// BuildSTT returns the configured speech-to-text client.
func BuildSTT(cfg config.InferenceConfig, provider *iam.Provider) providers.STTClient {
	switch strings.ToLower(cfg.STTProvider) {
	case "mock":
		return providers.NewMockSTT(os.Getenv("MOCK_STT_TRANSCRIPT"))
	default:
		return providers.NewYandexSTT(cfg.FolderID, cfg.STTModel, provider, cfg.STTPollInterval, cfg.STTPollTimeout)
	}
}

// BuildLLM returns the configured LLM client.
func BuildLLM(cfg config.InferenceConfig, provider *iam.Provider) providers.LLMClient {
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
