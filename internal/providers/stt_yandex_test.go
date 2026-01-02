package providers

import (
	"strings"
	"testing"
)

func TestDecodeRecognitionTranscriptMultiDoc(t *testing.T) {
	input := `{"result":{"final":{"alternatives":[{"text":"hello"}]}}}
{"result":{"final":{"alternatives":[{"text":"world"}]}}}`

	text, err := decodeRecognitionTranscript(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello world" {
		t.Fatalf("unexpected transcript: %q", text)
	}
}
