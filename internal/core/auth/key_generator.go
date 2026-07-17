package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRawKey(t APIKeyType) (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	prefix := "sk-personal-"

	if t == KeyTypeServiceAccount {
		prefix = "sk-project-"
	}

	return prefix + hex.EncodeToString(b), nil
}
