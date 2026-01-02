package hash

import (
	"testing"
	"time"
)

func TestNormalizeString(t *testing.T) {
	value := "  hello\nworld\t  foo  "
	expected := "hello world foo"
	if got := NormalizeString(value); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestBuildDialogHashUsesBucketWhenNoDialogID(t *testing.T) {
	base := DialogHashInput{
		APIKeyHash: "a",
		UserHash:   "u",
		PromptHash: "p",
		AudioHash:  "aud",
		Language:   "ru-RU",
		Persona:    "",
		DialogID:   "",
		StepID:     "",
		WindowSec:  60,
	}

	first := BuildDialogHash(DialogHashInput{Now: time.Unix(0, 0), APIKeyHash: base.APIKeyHash, UserHash: base.UserHash, PromptHash: base.PromptHash, AudioHash: base.AudioHash, Language: base.Language, Persona: base.Persona, DialogID: base.DialogID, StepID: base.StepID, WindowSec: base.WindowSec})
	second := BuildDialogHash(DialogHashInput{Now: time.Unix(120, 0), APIKeyHash: base.APIKeyHash, UserHash: base.UserHash, PromptHash: base.PromptHash, AudioHash: base.AudioHash, Language: base.Language, Persona: base.Persona, DialogID: base.DialogID, StepID: base.StepID, WindowSec: base.WindowSec})

	if first == second {
		t.Fatalf("expected different hashes for different buckets")
	}
}

func TestBuildDialogHashIgnoresBucketWhenDialogIDSet(t *testing.T) {
	base := DialogHashInput{
		APIKeyHash: "a",
		UserHash:   "u",
		PromptHash: "p",
		AudioHash:  "aud",
		Language:   "ru-RU",
		Persona:    "",
		DialogID:   "dlg-1",
		StepID:     "1",
		WindowSec:  60,
	}

	first := BuildDialogHash(DialogHashInput{Now: time.Unix(0, 0), APIKeyHash: base.APIKeyHash, UserHash: base.UserHash, PromptHash: base.PromptHash, AudioHash: base.AudioHash, Language: base.Language, Persona: base.Persona, DialogID: base.DialogID, StepID: base.StepID, WindowSec: base.WindowSec})
	second := BuildDialogHash(DialogHashInput{Now: time.Unix(120, 0), APIKeyHash: base.APIKeyHash, UserHash: base.UserHash, PromptHash: base.PromptHash, AudioHash: base.AudioHash, Language: base.Language, Persona: base.Persona, DialogID: base.DialogID, StepID: base.StepID, WindowSec: base.WindowSec})

	if first != second {
		t.Fatalf("expected same hash for same dialog id")
	}
}
