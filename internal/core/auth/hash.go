package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

type Hasher interface {
	Hash(string) string
}

type SHA256Hasher struct{}

var _ Hasher = SHA256Hasher{}

func NewSHA256Hasher() Hasher {
	return SHA256Hasher{}
}

func (s SHA256Hasher) Hash(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
