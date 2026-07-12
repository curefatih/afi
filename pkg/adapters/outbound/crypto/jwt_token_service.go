package crypto

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
	"github.com/golang-jwt/jwt/v5"
)

type JWTTokenService struct {
	secretKey     []byte
	issuer        string
	tokenDuration time.Duration
}

// UserClaims defines the structured payload contained inside the signed JWT token block
type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

var _ ports.PlatformTokenService = &JWTTokenService{}

// NewJWTTokenService constructs a standard cryptographically signed session manager
func NewJWTTokenService(secret string, issuer string, duration time.Duration) *JWTTokenService {
	return &JWTTokenService{
		secretKey:     []byte(secret),
		issuer:        issuer,
		tokenDuration: duration,
	}
}

// GenerateToken wraps user identity components into a signed JWT string
func (s *JWTTokenService) GenerateToken(ctx context.Context, user *domain.PlatformUser) (string, error) {
	now := time.Now()
	claims := UserClaims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			Issuer:    s.issuer,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("jwt generation token signing failed: %w", err)
	}

	return signedString, nil
}

// ValidateToken decodes, parses, and cryptographically verifies the raw token string
func (s *JWTTokenService) ValidateToken(ctx context.Context, tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Enforce validation of symmetric cryptographic algorithm integrity
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected parsing cryptographic signing algorithm: %v", t.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", errors.New("provided administrative session token has expired")
		}
		return "", fmt.Errorf("invalid token verification sequence: %w", err)
	}

	// Unpack explicit claims context references
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		if claims.UserID == "" {
			return "", errors.New("token verification passed but missing user identification metadata identifier")
		}
		return claims.UserID, nil
	}

	return "", errors.New("untrusted session token structure")
}

func (s *JWTTokenService) GetUserPermissions(ctx context.Context, userID string, orgID string, projectID string) ([]domain.ActionPermission, error) {
	// give all permissions for now
	permissions := []domain.ActionPermission{
		domain.PermOrgUserRead,
		domain.PermOrgUserWrite,
		domain.PermOrgRoleWrite,
		domain.PermProjectCreate,
		domain.PermOrgSpendRead,
		domain.PermProjectKeyWrite,
		domain.PermProjectKeyRead,
		domain.PermModelRouteWrite,
		domain.PermLLMInference,
	}
	return permissions, nil
}
