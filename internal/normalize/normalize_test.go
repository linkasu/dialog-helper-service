package normalize

import "testing"

func TestExtractTextsFromMessageJSON(t *testing.T) {
	message := "{\"texts\": [\"one\", \"two\"]}"
	texts := ExtractTextsFromMessage(message)
	if len(texts) != 2 {
		t.Fatalf("expected 2 texts, got %d", len(texts))
	}
	if texts[0] != "one" || texts[1] != "two" {
		t.Fatalf("unexpected texts: %#v", texts)
	}
}

func TestExtractTextsFromMessageCodeFence(t *testing.T) {
	message := "```json\n{\"texts\": [\"alpha\"]}\n```"
	texts := ExtractTextsFromMessage(message)
	if len(texts) != 1 || texts[0] != "alpha" {
		t.Fatalf("unexpected texts: %#v", texts)
	}
}

func TestNormalizeCandidates(t *testing.T) {
	texts := []string{" Hello ", "hello", "world", ""}
	candidates := NormalizeCandidates(texts, 2, "ru-RU")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].Text != "Hello" || candidates[1].Text != "world" {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
}
