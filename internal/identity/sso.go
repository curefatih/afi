package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

// ErrEmailNotVerified is returned when JIT provisioning requires a verified email.
var ErrEmailNotVerified = errors.New("email not verified")

// ErrMissingFederatedEmail is returned when claims lack an email address.
var ErrMissingFederatedEmail = errors.New("federated email required")

// ErrMissingFederatedSubject is returned when claims lack a subject.
var ErrMissingFederatedSubject = errors.New("federated subject required")

// ResolveDeps are persistence ports for ResolveOrProvisionUser.
type ResolveDeps struct {
	Users      UserRepository
	Identities ExternalIdentityRepository
	NewUserID  func() string
	NewLinkID  func() string
	Now        func() time.Time
}

// ResolveOrProvisionUser finds or creates a platform user from federated claims (JIT).
//
// Resolution order:
//  1. Existing ExternalIdentity by (provider, subject)
//  2. Verified email matches existing user → link identity
//  3. JIT-create user (role member, no password) + link identity
//
// Unverified email is rejected when no prior link exists.
func ResolveOrProvisionUser(ctx context.Context, deps ResolveDeps, claims FederatedClaims) (*User, error) {
	if deps.Users == nil || deps.Identities == nil {
		return nil, fmt.Errorf("identity resolve: missing repositories")
	}
	claims.Provider = strings.TrimSpace(claims.Provider)
	claims.Subject = strings.TrimSpace(claims.Subject)
	claims.Email = normalizeEmail(claims.Email)
	claims.Name = strings.TrimSpace(claims.Name)
	claims.Issuer = strings.TrimSpace(claims.Issuer)

	if claims.Provider == "" || claims.Subject == "" {
		return nil, ErrMissingFederatedSubject
	}

	now := time.Now().UTC()
	if deps.Now != nil {
		now = deps.Now().UTC()
	}
	newUserID := deps.NewUserID
	if newUserID == nil {
		return nil, fmt.Errorf("identity resolve: NewUserID required")
	}
	newLinkID := deps.NewLinkID
	if newLinkID == nil {
		return nil, fmt.Errorf("identity resolve: NewLinkID required")
	}

	if link, err := deps.Identities.GetByProviderSubject(ctx, claims.Provider, claims.Subject); err == nil {
		return deps.Users.GetByID(ctx, link.UserID)
	} else if !errors.Is(err, kernel.ErrNotFound) {
		return nil, err
	}

	if claims.Email == "" {
		return nil, ErrMissingFederatedEmail
	}
	if !claims.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	existing, err := deps.Users.GetByEmail(ctx, claims.Email)
	if err == nil {
		link := ExternalIdentity{
			ID:        newLinkID(),
			UserID:    existing.ID,
			Provider:  claims.Provider,
			Issuer:    claims.Issuer,
			Subject:   claims.Subject,
			Email:     claims.Email,
			CreatedAt: now,
		}
		if err := deps.Identities.Create(ctx, link); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, kernel.ErrNotFound) {
		return nil, err
	}

	name := claims.Name
	if name == "" {
		name = claims.Email
	}
	user := User{
		ID:           newUserID(),
		Email:        claims.Email,
		Name:         name,
		Role:         "member",
		PasswordHash: "",
		CreatedAt:    now,
	}
	if err := deps.Users.Create(ctx, user); err != nil {
		return nil, err
	}
	link := ExternalIdentity{
		ID:        newLinkID(),
		UserID:    user.ID,
		Provider:  claims.Provider,
		Issuer:    claims.Issuer,
		Subject:   claims.Subject,
		Email:     claims.Email,
		CreatedAt: now,
	}
	if err := deps.Identities.Create(ctx, link); err != nil {
		return nil, err
	}
	return &user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
