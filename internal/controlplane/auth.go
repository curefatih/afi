package controlplane

import (
	"time"

	"github.com/curefatih/afi/internal/adapters/auth"
)

type Claims = auth.Claims

func HashPassword(password string) (string, error) {
	return auth.HashPassword(password)
}

func CheckPassword(hash, password string) bool {
	return auth.CheckPassword(hash, password)
}

func IssueToken(secret string, ttl time.Duration, userID, email, role string) (string, error) {
	return auth.IssueToken(secret, ttl, userID, email, role)
}

func ParseToken(secret, token string) (*Claims, error) {
	return auth.ParseToken(secret, token)
}
