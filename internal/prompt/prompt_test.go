package prompt

import (
	"testing"

	"github.com/linkasu/dialog-helper-service/internal/models"
)

func TestBuildUserMessage(t *testing.T) {
	messages := []models.DialogMessage{
		{Role: "speaker", Content: "hello"},
		{Role: "disabled_person", Content: "hi"},
	}
	message := BuildUserMessage("client context", messages, "hello")
	expected := "Disabled person biography:\nclient context\n\nDialog:\nspeaker: hello\ndisabled_person: hi\n\nTranscript (speaker audio):\nhello"
	if message != expected {
		t.Fatalf("expected %q, got %q", expected, message)
	}
}

func TestBuildUserMessageEmpty(t *testing.T) {
	message := BuildUserMessage("", nil, "")
	if message != "" {
		t.Fatalf("expected empty message, got %q", message)
	}
}
