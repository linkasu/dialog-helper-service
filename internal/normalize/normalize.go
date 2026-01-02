package normalize

import (
	"encoding/json"
	"strings"

	"github.com/linkasu/dialog-helper-service/internal/models"
)

const defaultMaxCandidates = 5

func ExtractTextsFromMessage(message string) []string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return nil
	}

	cleaned := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "```json"), "```"))
	cleaned = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(cleaned, "```"), "```"))

	var parsed any
	if err := json.Unmarshal([]byte(cleaned), &parsed); err == nil {
		return extractTexts(parsed, 0)
	}

	return []string{trimmed}
}

func extractTexts(value any, depth int) []string {
	if value == nil || depth > 6 {
		return nil
	}

	switch v := value.(type) {
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return nil
		}
		var nested any
		if err := json.Unmarshal([]byte(text), &nested); err == nil {
			return extractTexts(nested, depth+1)
		}
		return []string{text}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, extractTexts(item, depth+1)...)
		}
		return out
	case map[string]any:
		if raw, ok := v["texts"]; ok {
			return extractTexts(raw, depth+1)
		}
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, extractTexts(item, depth+1)...)
		}
		return out
	default:
		return nil
	}
}

func NormalizeCandidates(texts []string, max int, language string) []models.AnswerCandidate {
	limit := max
	if limit <= 0 {
		limit = defaultMaxCandidates
	}

	seen := make(map[string]struct{})
	candidates := make([]models.AnswerCandidate, 0, limit)
	for _, raw := range texts {
		cleaned := strings.TrimSpace(raw)
		if cleaned == "" {
			continue
		}
		key := strings.ToLower(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		candidates = append(candidates, models.AnswerCandidate{
			Text:     cleaned,
			Rank:     len(candidates) + 1,
			Length:   len([]rune(cleaned)),
			Language: language,
		})
		if len(candidates) >= limit {
			break
		}
	}

	return candidates
}
