package prompt

import "testing"

func TestBuildUserMessage(t *testing.T) {
	message := BuildUserMessage("client context", "hello")
	expected := "Client prompt:\nclient context\n\nTranscript:\nhello"
	if message != expected {
		t.Fatalf("expected %q, got %q", expected, message)
	}
}

func TestBuildUserMessageEmpty(t *testing.T) {
	message := BuildUserMessage("", "")
	if message != "" {
		t.Fatalf("expected empty message, got %q", message)
	}
}
