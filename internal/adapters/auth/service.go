package auth

import (
	"time"

	"github.com/curefatih/afi/internal/identity"
)

// Service implements identity.TokenIssuer and identity.PasswordHasher.
type Service struct {
	JWTSecret string
	TokenTTL  time.Duration
}

func NewService(jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{JWTSecret: jwtSecret, TokenTTL: tokenTTL}
}

func (s *Service) Issue(userID, email, role string) (string, error) {
	return IssueToken(s.JWTSecret, s.TokenTTL, userID, email, role)
}

func (s *Service) Hash(password string) (string, error) {
	return HashPassword(password)
}

func (s *Service) Check(hash, password string) bool {
	if hash == "" {
		return false
	}
	return CheckPassword(hash, password)
}

var (
	_ identity.TokenIssuer    = (*Service)(nil)
	_ identity.PasswordHasher = (*Service)(nil)
)
