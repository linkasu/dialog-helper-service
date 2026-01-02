package providers

import "context"

type MockSTT struct {
	transcript string
}

func NewMockSTT(transcript string) *MockSTT {
	if transcript == "" {
		transcript = "mock transcript"
	}
	return &MockSTT{transcript: transcript}
}

func (m *MockSTT) Transcribe(_ context.Context, _ STTRequest) (STTResult, error) {
	return STTResult{Transcript: m.transcript}, nil
}
