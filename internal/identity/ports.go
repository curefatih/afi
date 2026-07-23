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
// OAuth2/OIDC and SAML 2.0 implement this port. SAML ACS maps the assertion
// payload at the HTTP edge into Exchange / AssertionExchanger.
type FederationProvider interface {
	// Name returns the configured provider id (e.g. "google").
	Name() string
	// DisplayName is a human-readable label for the login UI.
	DisplayName() string
	// Type is the protocol kind: "oidc", "oauth2", or "saml".
	Type() string
	// AuthURL builds the IdP authorization redirect URL for the given CSRF state.
	AuthURL(state, redirectURI string) (string, error)
	// Exchange trades an authorization code (or mapped SAMLResponse) for FederatedClaims.
	Exchange(ctx context.Context, code, redirectURI string) (FederatedClaims, error)
}

// SSOState holds short-lived CSRF state for an SSO login attempt.
type SSOState struct {
	Provider  string
	ReturnTo  string
	RequestID string // SAML AuthnRequest ID (empty for OAuth/OIDC)
	ExpiresAt time.Time
}

// SSOStateStore stores CSRF state between begin and complete SSO.
// Implementations must be safe for multi-instance control planes (shared backend).
type SSOStateStore interface {
	Put(ctx context.Context, state string, value SSOState) error
	// Take atomically loads and deletes state. Returns kernel.ErrNotFound when missing/expired.
	Take(ctx context.Context, state string) (SSOState, error)
}

// AuthStarter is an optional FederationProvider extension used by SAML to return
// the AuthnRequest ID for InResponseTo validation on ACS.
type AuthStarter interface {
	AuthURLWithID(state, redirectURI string) (authURL, requestID string, err error)
}

// AssertionExchanger is an optional FederationProvider extension for SAML ACS.
// possibleRequestIDs should include the AuthnRequest ID from BeginSSO (and "" when IdP-initiated is allowed).
type AssertionExchanger interface {
	ExchangeAssertion(ctx context.Context, response, redirectURI string, possibleRequestIDs []string) (FederatedClaims, error)
}

// ServiceProviderMeta is implemented by SAML adapters that expose SP metadata XML.
type ServiceProviderMeta interface {
	MetadataXML() ([]byte, error)
}
