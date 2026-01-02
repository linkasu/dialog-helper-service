package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const defaultTimeWindowSec = 600

func SHA256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func SHA256HexString(value string) string {
	return SHA256HexBytes([]byte(value))
}

func NormalizeString(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.ReplaceAll(trimmed, "\t", " ")
	return strings.Join(strings.Fields(trimmed), " ")
}

func HashAPIKey(key string) string {
	return SHA256HexString(NormalizeString(key))
}

func HashPrompt(prompt string) string {
	return SHA256HexString(NormalizeString(prompt))
}

func HashUser(userID, userProfile string) string {
	payload := NormalizeString(userID) + ":" + NormalizeString(userProfile)
	return SHA256HexString(payload)
}

type DialogHashInput struct {
	APIKeyHash string
	UserHash   string
	PromptHash string
	AudioHash  string
	Language   string
	Persona    string
	DialogID   string
	StepID     string
	Now        time.Time
	WindowSec  int
}

func BuildDialogHash(input DialogHashInput) string {
	window := input.WindowSec
	if window <= 0 {
		window = defaultTimeWindowSec
	}

	bucket := timeBucket(input.Now.UTC(), window)
	parts := []string{
		"api=" + NormalizeString(input.APIKeyHash),
		"user=" + NormalizeString(input.UserHash),
		"prompt=" + NormalizeString(input.PromptHash),
		"audio=" + NormalizeString(input.AudioHash),
		"lang=" + NormalizeString(input.Language),
		"persona=" + NormalizeString(input.Persona),
		"dialog=" + NormalizeString(input.DialogID),
		"step=" + NormalizeString(input.StepID),
	}

	if input.DialogID == "" {
		parts = append(parts, "bucket="+bucket)
	}

	return SHA256HexString(strings.Join(parts, "|"))
}

func timeBucket(now time.Time, windowSec int) string {
	if windowSec <= 0 {
		windowSec = defaultTimeWindowSec
	}
	epoch := now.Unix()
	bucketStart := epoch - (epoch % int64(windowSec))
	return time.Unix(bucketStart, 0).UTC().Format("20060102150405")
}
