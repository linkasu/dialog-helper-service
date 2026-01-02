package providers

import "context"

type STTRequest struct {
	Audio              []byte
	ContainerAudioType string
	Language           string
}

type STTResult struct {
	Transcript string
}

type STTClient interface {
	Transcribe(ctx context.Context, req STTRequest) (STTResult, error)
}
