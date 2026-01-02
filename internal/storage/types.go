package storage

import (
	"time"

	"github.com/linkasu/dialog-helper-service/internal/models"
)

type ResultRecord struct {
	DialogHash string
	UserHash   string
	Transcript string
	Candidates []models.AnswerCandidate
	MetaJSON   string
	CreatedAt  time.Time
}

type APIKeyRecord struct {
	APIKeyHash         string
	ClientID           string
	Status             string
	Scopes             []string
	RateLimitPerMinute int
	CreatedAt          time.Time
	RevokedAt          *time.Time
}

type ConfigItem struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}
