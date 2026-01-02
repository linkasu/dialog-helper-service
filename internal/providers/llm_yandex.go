package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/iam"
	"github.com/linkasu/dialog-helper-service/internal/normalize"
)

const yandexLLMEndpoint = "https://llm.api.cloud.yandex.net/foundationModels/v1/completion"

type YandexLLM struct {
	folderID string
	provider *iam.Provider
	client   *http.Client
}

func NewYandexLLM(folderID string, provider *iam.Provider) *YandexLLM {
	return &YandexLLM{
		folderID: folderID,
		provider: provider,
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func (y *YandexLLM) Complete(ctx context.Context, req LLMRequest) (LLMResult, error) {
	if req.UserPrompt == "" {
		return LLMResult{}, errors.New("user prompt is required")
	}

	modelURI := req.ModelURI
	if modelURI == "" {
		modelURI = fmt.Sprintf("gpt://%s/yandexgpt/latest", y.folderID)
	}

	token, err := y.provider.Token(ctx)
	if err != nil {
		return LLMResult{}, err
	}

	body := map[string]interface{}{
		"modelUri": modelURI,
		"completionOptions": map[string]interface{}{
			"stream":      false,
			"temperature": req.Temperature,
			"maxTokens":   fmt.Sprintf("%d", req.MaxTokens),
		},
		"messages": []map[string]string{
			{"role": "system", "text": req.SystemPrompt},
			{"role": "user", "text": req.UserPrompt},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return LLMResult{}, err
	}

	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, yandexLLMEndpoint, bytes.NewReader(payload))
	if err != nil {
		return LLMResult{}, err
	}
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	reqHTTP.Header.Set("x-folder-id", y.folderID)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := y.client.Do(reqHTTP)
	if err != nil {
		return LLMResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return LLMResult{}, errors.New("llm request failed")
	}

	var decoded struct {
		Result struct {
			Alternatives []struct {
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
			} `json:"alternatives"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return LLMResult{}, err
	}

	message := ""
	if len(decoded.Result.Alternatives) > 0 {
		message = decoded.Result.Alternatives[0].Message.Text
	}

	texts := normalize.ExtractTextsFromMessage(message)
	return LLMResult{Message: message, Texts: texts}, nil
}
