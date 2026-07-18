package auth

import (
	"crypto/rand"
	"encoding/hex"
)

type KeyGenerator interface {
	Generate(APIKeyType) (string, error)
}

type DefaultKeyGenerator struct {
}

var _ KeyGenerator = DefaultKeyGenerator{}

func NewKeyGenerator() KeyGenerator {
	return DefaultKeyGenerator{}
}

func (d DefaultKeyGenerator) Generate(t APIKeyType) (string, error) {
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
