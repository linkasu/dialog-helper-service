package models

import "time"

type InferPayload struct {
	Prompt        string        `json:"prompt"`
	Language      string        `json:"language,omitempty"`
	Persona       string        `json:"persona,omitempty"`
	UserID        string        `json:"user_id,omitempty"`
	UserProfile   string        `json:"user_profile,omitempty"`
	DialogID      string        `json:"dialog_id,omitempty"`
	StepID        string        `json:"step_id,omitempty"`
	TimeWindowSec int           `json:"time_window_sec,omitempty"`
	Transcript    string        `json:"transcript,omitempty"`
	Options       *InferOptions `json:"options,omitempty"`
}

type InferOptions struct {
	MaxCandidates int     `json:"max_candidates,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
}

type InferResponse struct {
	RequestID  string            `json:"request_id"`
	DialogHash string            `json:"dialog_hash"`
	UserHash   string            `json:"user_hash"`
	Transcript TranscriptResult  `json:"transcript"`
	Candidates []AnswerCandidate `json:"candidates"`
	Errors     []APIError        `json:"errors,omitempty"`
	Meta       Meta              `json:"meta"`
}

type TranscriptResult struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

type AnswerCandidate struct {
	Text       string   `json:"text"`
	Rank       int      `json:"rank"`
	Length     int      `json:"length"`
	Language   string   `json:"language,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type Meta struct {
	CacheHit            bool   `json:"cache_hit"`
	STTLatencyMs        int64  `json:"stt_latency_ms,omitempty"`
	LLMLatencyMs        int64  `json:"llm_latency_ms,omitempty"`
	SystemPromptVersion string `json:"system_prompt_version,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIKeyInfo struct {
	ClientID           string   `json:"client_id"`
	Status             string   `json:"status"`
	Scopes             []string `json:"scopes,omitempty"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute,omitempty"`
}

type APIKeyCreateRequest struct {
	ClientID           string   `json:"client_id"`
	Scopes             []string `json:"scopes,omitempty"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute,omitempty"`
	Status             string   `json:"status,omitempty"`
}

type APIKeyCreateResponse struct {
	APIKey             string    `json:"api_key"`
	APIKeyHash         string    `json:"api_key_hash"`
	ClientID           string    `json:"client_id"`
	Status             string    `json:"status"`
	Scopes             []string  `json:"scopes,omitempty"`
	RateLimitPerMinute int       `json:"rate_limit_per_minute,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}
