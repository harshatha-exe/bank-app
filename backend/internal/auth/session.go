package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

func newSessionID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func sessionHash(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func csrfToken(sessionID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Username     string
	Role         string
	LastActiveAt time.Time
	ExpiresAt    time.Time
}
