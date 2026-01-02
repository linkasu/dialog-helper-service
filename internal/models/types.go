package models

import "time"

type InferPayload struct {
	DisabledPersonBiography string          `json:"disabled_person_biography,omitempty"`
	Messages                []DialogMessage `json:"messages"`
	Language                string          `json:"language,omitempty"`
	Persona                 string          `json:"persona,omitempty"`
	UserID                  string          `json:"user_id,omitempty"`
	UserProfile             string          `json:"user_profile,omitempty"`
	DialogID                string          `json:"dialog_id,omitempty"`
	StepID                  string          `json:"step_id,omitempty"`
	TimeWindowSec           int             `json:"time_window_sec,omitempty"`
	Options                 *InferOptions   `json:"options,omitempty"`
}

type InferOptions struct {
	MaxCandidates int     `json:"max_candidates,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
}

type InferResponse struct {
	Transcript *string  `json:"transcript,omitempty"`
	Response   []string `json:"response"`
}

type AnswerCandidate struct {
	Text       string   `json:"text"`
	Rank       int      `json:"rank"`
	Length     int      `json:"length"`
	Language   string   `json:"language,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type DialogMessage struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
	Audio   bool   `json:"audio,omitempty"`
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
