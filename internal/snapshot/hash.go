package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashKey returns a hex-encoded SHA-256 of the raw virtual API key.
func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
