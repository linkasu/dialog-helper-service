package prompt

import (
	"strings"
)

func BuildUserMessage(clientPrompt, transcript string) string {
	client := strings.TrimSpace(clientPrompt)
	trans := strings.TrimSpace(transcript)

	sections := make([]string, 0, 3)
	if client != "" {
		sections = append(sections, "Client prompt:\n"+client)
	}
	if trans != "" {
		sections = append(sections, "Transcript:\n"+trans)
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

func NormalizeSystemPrompt(systemPrompt string) string {
	trimmed := strings.TrimSpace(systemPrompt)
	return trimmed
}
