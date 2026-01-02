package prompt

import (
	"strings"

	"github.com/linkasu/dialog-helper-service/internal/models"
)

func BuildUserMessage(biography string, messages []models.DialogMessage, transcript string) string {
	bio := strings.TrimSpace(biography)
	trans := strings.TrimSpace(transcript)

	sections := make([]string, 0, 4)
	if bio != "" {
		sections = append(sections, "Disabled person biography:\n"+bio)
	}

	dialogLines := make([]string, 0, len(messages))
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if role == "" || content == "" {
			continue
		}
		dialogLines = append(dialogLines, role+": "+content)
	}
	if len(dialogLines) > 0 {
		sections = append(sections, "Dialog:\n"+strings.Join(dialogLines, "\n"))
	}
	if trans != "" {
		sections = append(sections, "Transcript (speaker audio):\n"+trans)
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
