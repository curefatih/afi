package identity

import (
	"context"
	"time"
)

// UserRepository loads and creates platform users.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, user User) error
}

// ExternalIdentityRepository persists federated IdP linkages.
type ExternalIdentityRepository interface {
	GetByProviderSubject(ctx context.Context, provider, subject string) (*ExternalIdentity, error)
	Create(ctx context.Context, identity ExternalIdentity) error
}

// TokenIssuer creates platform session JWTs.
type TokenIssuer interface {
	Issue(userID, email, role string) (string, error)
}

// PasswordHasher hashes and verifies local passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Check(hash, password string) bool
}

// FederationProvider is a protocol-agnostic SSO IdP adapter.
// OAuth2/OIDC implement this now; SAML 2.0 can implement the same port later
// (ACS would call Exchange with an assertion payload mapped at the HTTP edge).
type FederationProvider interface {
	// Name returns the configured provider id (e.g. "google").
	Name() string
	// DisplayName is a human-readable label for the login UI.
	DisplayName() string
	// Type is the protocol kind: "oidc", "oauth2", or later "saml".
	Type() string
	// AuthURL builds the IdP authorization redirect URL for the given CSRF state.
	AuthURL(state, redirectURI string) (string, error)
	// Exchange trades an authorization code for FederatedClaims.
	Exchange(ctx context.Context, code, redirectURI string) (FederatedClaims, error)
}

// SSOState holds short-lived CSRF state for an SSO login attempt.
type SSOState struct {
	Provider  string
	ReturnTo  string
	ExpiresAt time.Time
}

// SSOStateStore stores CSRF state between begin and complete SSO.
// Implementations must be safe for multi-instance control planes (shared backend).
type SSOStateStore interface {
	Put(ctx context.Context, state string, value SSOState) error
	// Take atomically loads and deletes state. Returns kernel.ErrNotFound when missing/expired.
	Take(ctx context.Context, state string) (SSOState, error)
}
