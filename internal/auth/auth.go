package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"

	"github.com/linkasu/dialog-helper-service/internal/hash"
)

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusRevoked   = "revoked"
)

const apiKeyPrefix = "dhk_"

func GenerateAPIKey() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate api key: %w", err)
	}

	plain := apiKeyPrefix + hex.EncodeToString(buf)
	hashed := hash.HashAPIKey(plain)
	return plain, hashed, nil
}

func HashAPIKey(key string) string {
	return hash.HashAPIKey(key)
}

func ConstantTimeEquals(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
