package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashKey(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
