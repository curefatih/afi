package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/curefatih/afi/internal/core/domain"
)

var ErrUnauthorized = errors.New("invalid or deactivated token credential")

type AuthRepository interface {
	SaveAPIKey(ctx context.Context, apiKey *domain.APIKey) error
	GetContextByKeyHash(ctx context.Context, hash string) (*domain.RequestContext, error)
}

type AuthService struct {
	repo AuthRepository
}

func NewAuthService(repo AuthRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) IssueAPIKey(ctx context.Context, keyType domain.APIKeyType, targetID string) (string, error) {
	// 1. Generate secure random key token string
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	prefix := "sk-personal-"
	if keyType == domain.KeyTypeServiceAccount {
		prefix = "sk-project-"
	}
	rawKey := prefix + hex.EncodeToString(b)

	// 2. Hash it securely using SHA-256 before writing to DB
	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := hex.EncodeToString(hash[:])

	apiKey := &domain.APIKey{
		HashedKey: hashedKey,
		Type:      keyType,
	}

	if keyType == domain.KeyTypePersonal {
		apiKey.UserID = targetID
	} else {
		apiKey.ProjectID = targetID
	}

	if err := s.repo.SaveAPIKey(ctx, apiKey); err != nil {
		return "", err
	}

	return rawKey, nil
}

func (s *AuthService) AuthenticateKey(ctx context.Context, rawKey string) (*domain.RequestContext, error) {
	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := hex.EncodeToString(hash[:])

	// Fetch unified metadata mapping using the hashed representation
	reqContext, err := s.repo.GetContextByKeyHash(ctx, hashedKey)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return reqContext, nil
}
