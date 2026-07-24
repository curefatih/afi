package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const DefaultPasswordResetTTL = time.Hour

// PasswordResetToken is a stored reset challenge (hash only; raw token is emailed).
type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time
}

// HashOpaqueToken returns the hex-encoded SHA-256 of the raw token.
func HashOpaqueToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewOpaqueToken generates an opaque token and its hash (same shape as invite tokens).
func NewOpaqueToken() (raw, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b[:])
	return raw, HashOpaqueToken(raw), nil
}
