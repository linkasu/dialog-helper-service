package providers

import "context"

type LLMRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
	ModelURI     string
}

type LLMResult struct {
	Message string
	Texts   []string
}

type LLMClient interface {
	Complete(ctx context.Context, req LLMRequest) (LLMResult, error)
}
