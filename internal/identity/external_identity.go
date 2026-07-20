package identity

import "time"

// ExternalIdentity links a platform user to a federated IdP subject.
// Uniqueness is (Provider, Subject).
type ExternalIdentity struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Provider  string    `json:"provider"`
	Issuer    string    `json:"issuer"`
	Subject   string    `json:"subject"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// FederatedClaims are protocol-agnostic identity assertions from an IdP.
// OAuth2, OIDC, and (later) SAML adapters all map into this shape.
type FederatedClaims struct {
	Provider      string
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}
