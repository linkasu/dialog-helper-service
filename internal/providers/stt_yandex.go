package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/iam"
)

const yandexSTTEndpoint = "https://stt.api.cloud.yandex.net/stt/v3/recognizeFile"

type YandexSTT struct {
	folderID string
	model    string
	provider *iam.Provider
	client   *http.Client
}

func NewYandexSTT(folderID, model string, provider *iam.Provider) *YandexSTT {
	return &YandexSTT{
		folderID: folderID,
		model:    model,
		provider: provider,
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func (y *YandexSTT) Transcribe(ctx context.Context, req STTRequest) (STTResult, error) {
	if len(req.Audio) == 0 {
		return STTResult{}, errors.New("audio is required")
	}
	if req.ContainerAudioType == "" {
		return STTResult{}, errors.New("container audio type is required")
	}

	token, err := y.provider.Token(ctx)
	if err != nil {
		return STTResult{}, err
	}

	body := map[string]interface{}{
		"content": base64.StdEncoding.EncodeToString(req.Audio),
		"recognitionModel": map[string]interface{}{
			"model": y.model,
			"audioFormat": map[string]interface{}{
				"containerAudio": map[string]interface{}{
					"containerAudioType": req.ContainerAudioType,
				},
			},
			"languageRestriction": map[string]interface{}{
				"restrictionType": "WHITELIST",
				"languageCode":    []string{req.Language},
			},
			"textNormalization": map[string]interface{}{
				"textNormalization": "TEXT_NORMALIZATION_DISABLED",
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return STTResult{}, err
	}

	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, yandexSTTEndpoint, bytes.NewReader(payload))
	if err != nil {
		return STTResult{}, err
	}
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	reqHTTP.Header.Set("x-folder-id", y.folderID)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := y.client.Do(reqHTTP)
	if err != nil {
		return STTResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return STTResult{}, errors.New("stt request failed")
	}

	var decoded interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return STTResult{}, err
	}

	transcript := extractTranscript(decoded)
	return STTResult{Transcript: transcript}, nil
}

func extractTranscript(data interface{}) string {
	if data == nil {
		return ""
	}

	switch value := data.(type) {
	case []interface{}:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			text := extractTranscript(item)
			if text != "" {
				parts = append(parts, text)
			}
		}
		return joinParts(parts)
	case map[string]interface{}:
		if text, ok := value["text"].(string); ok && text != "" {
			return text
		}
		if text, ok := value["normalizedText"].(string); ok && text != "" {
			return text
		}
		if alternatives, ok := value["alternatives"].([]interface{}); ok && len(alternatives) > 0 {
			if first, ok := alternatives[0].(map[string]interface{}); ok {
				if text, ok := first["text"].(string); ok && text != "" {
					return text
				}
			}
		}
		result, _ := value["result"].(map[string]interface{})
		candidates := []interface{}{
			result["final"],
			result["chunks"],
			result["segments"],
			value["final"],
			result["text"],
			value["text"],
		}
		parts := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			parts = append(parts, extractTranscript(candidate))
		}
		return joinParts(parts)
	case string:
		return value
	default:
		return ""
	}
}

func joinParts(parts []string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return strings.Join(clean, " ")
}
