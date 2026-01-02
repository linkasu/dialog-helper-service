package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/iam"
)

const (
	yandexSTTEndpoint         = "https://stt.api.cloud.yandex.net/stt/v3/recognizeFileAsync"
	yandexSTTOperationBaseURL = "https://operation.api.cloud.yandex.net/operations/"
	yandexSTTResultEndpoint   = "https://stt.api.cloud.yandex.net/stt/v3/getRecognition"
)

type YandexSTT struct {
	folderID     string
	model        string
	provider     *iam.Provider
	client       *http.Client
	pollInterval time.Duration
	pollTimeout  time.Duration
}

func NewYandexSTT(folderID, model string, provider *iam.Provider, pollInterval, pollTimeout time.Duration) *YandexSTT {
	if pollInterval <= 0 {
		pollInterval = 200 * time.Millisecond
	}
	if pollTimeout <= 0 {
		pollTimeout = 10 * time.Second
	}

	return &YandexSTT{
		folderID:     folderID,
		model:        model,
		provider:     provider,
		pollInterval: pollInterval,
		pollTimeout:  pollTimeout,
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

	operationID, err := y.startRecognition(ctx, token, req)
	if err != nil {
		return STTResult{}, err
	}

	if err := y.waitOperation(ctx, token, operationID); err != nil {
		return STTResult{}, err
	}

	transcript, err := y.fetchRecognition(ctx, token, operationID)
	if err != nil {
		return STTResult{}, err
	}

	return STTResult{Transcript: transcript}, nil
}

func (y *YandexSTT) startRecognition(ctx context.Context, token string, req STTRequest) (string, error) {
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
		return "", err
	}

	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, yandexSTTEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	reqHTTP.Header.Set("x-folder-id", y.folderID)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := y.client.Do(reqHTTP)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", readSTTError(resp, "stt request failed")
	}

	var payloadResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payloadResp); err != nil {
		return "", err
	}
	if payloadResp.ID == "" {
		return "", errors.New("stt response missing operation id")
	}
	return payloadResp.ID, nil
}

func (y *YandexSTT) waitOperation(ctx context.Context, token, operationID string) error {
	deadline := time.Now().Add(y.pollTimeout)
	for {
		if time.Now().After(deadline) {
			return errors.New("stt operation timeout")
		}

		reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodGet, yandexSTTOperationBaseURL+operationID, nil)
		if err != nil {
			return err
		}
		reqHTTP.Header.Set("Authorization", "Bearer "+token)

		resp, err := y.client.Do(reqHTTP)
		if err != nil {
			return err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			err := readSTTError(resp, "stt operation failed")
			resp.Body.Close()
			return err
		}

		var payload struct {
			Done  bool        `json:"done"`
			Error interface{} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		if payload.Done {
			if payload.Error != nil {
				return fmt.Errorf("stt operation error: %v", payload.Error)
			}
			return nil
		}

		timer := time.NewTimer(y.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (y *YandexSTT) fetchRecognition(ctx context.Context, token, operationID string) (string, error) {
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodGet, yandexSTTResultEndpoint+"?operation_id="+operationID, nil)
	if err != nil {
		return "", err
	}
	reqHTTP.Header.Set("Authorization", "Bearer "+token)

	resp, err := y.client.Do(reqHTTP)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", readSTTError(resp, "stt result failed")
	}

	return decodeRecognitionTranscript(resp.Body)
}

func decodeRecognitionTranscript(r io.Reader) (string, error) {
	decoder := json.NewDecoder(r)
	parts := make([]string, 0)
	for {
		var payload interface{}
		if err := decoder.Decode(&payload); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		text := extractTranscript(payload)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return joinParts(parts), nil
}

func readSTTError(resp *http.Response, fallback string) error {
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		return fmt.Errorf("%s: status %d", fallback, resp.StatusCode)
	}
	return fmt.Errorf("%s: status %d: %s", fallback, resp.StatusCode, strings.TrimSpace(string(body)))
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
