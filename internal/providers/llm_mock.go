package providers

import "context"

type MockLLM struct {
	texts []string
}

func NewMockLLM(texts []string) *MockLLM {
	if len(texts) == 0 {
		texts = []string{"Yes", "No", "Maybe"}
	}
	return &MockLLM{texts: texts}
}

func (m *MockLLM) Complete(_ context.Context, _ LLMRequest) (LLMResult, error) {
	return LLMResult{Texts: m.texts, Message: ""}, nil
}
