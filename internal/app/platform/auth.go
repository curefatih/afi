package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

// AuthService orchestrates password and SSO platform AuthN.
type AuthService struct {
	Users          identity.UserRepository
	Identities     identity.ExternalIdentityRepository
	Resets         identity.PasswordResetRepository
	Tokens         identity.TokenIssuer
	Passwords      identity.PasswordHasher
	States         identity.SSOStateStore
	Providers      map[string]identity.FederationProvider
	SSOEnabled     bool
	SignupEnabled  bool
	// PublicBaseURL is the control-plane base URL used for OAuth callbacks.
	PublicBaseURL string
	// AppBaseURL is the web UI base URL used after SSO callback redirect.
	AppBaseURL string
	NewUserID  func() string
	NewLinkID  func() string
	NewResetID func() string
	Now        func() time.Time
}

// AuthFeatures describes public auth capabilities for the login UI.
type AuthFeatures struct {
	SignupEnabled        bool `json:"signup_enabled"`
	PasswordResetEnabled bool `json:"password_reset_enabled"`
}

// SSOProviderInfo is a public descriptor for the login UI.
type SSOProviderInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

// ListSSOProviders returns enabled SSO providers for the login UI.
func (a *AuthService) ListSSOProviders() []SSOProviderInfo {
	if a == nil || !a.SSOEnabled || len(a.Providers) == 0 {
		return nil
	}
	out := make([]SSOProviderInfo, 0, len(a.Providers))
	for _, p := range a.Providers {
		out = append(out, SSOProviderInfo{
			ID:          p.Name(),
			DisplayName: p.DisplayName(),
			Type:        p.Type(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Features returns public auth capability flags for the UI.
func (a *AuthService) Features() AuthFeatures {
	if a == nil {
		return AuthFeatures{PasswordResetEnabled: true}
	}
	return AuthFeatures{
		SignupEnabled:        a.SignupEnabled,
		PasswordResetEnabled: true,
	}
}

// Register creates a local password user and returns a session JWT.
func (a *AuthService) Register(ctx context.Context, email, name, password string) (token string, user *identity.User, err error) {
	if a == nil || a.Users == nil || a.Passwords == nil || a.Tokens == nil {
		return "", nil, fmt.Errorf("auth service not configured")
	}
	if !a.SignupEnabled {
		return "", nil, identity.ErrSignupDisabled
	}
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if email == "" || !strings.Contains(email, "@") || name == "" {
		return "", nil, kernel.ErrInvalidRequest
	}
	if len(password) < 8 {
		return "", nil, kernel.ErrInvalidRequest
	}
	if _, err := a.Users.GetByEmail(ctx, email); err == nil {
		return "", nil, kernel.ErrConflict
	} else if !errors.Is(err, kernel.ErrNotFound) {
		return "", nil, err
	}
	hash, err := a.Passwords.Hash(password)
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	id := ""
	if a.NewUserID != nil {
		id = a.NewUserID()
	}
	if id == "" {
		return "", nil, fmt.Errorf("new user id not configured")
	}
	u := identity.User{
		ID:           id,
		Email:        email,
		Name:         name,
		Role:         "member",
		PasswordHash: hash,
		CreatedAt:    now,
	}
	if err := a.Users.Create(ctx, u); err != nil {
		return "", nil, err
	}
	tok, err := a.Tokens.Issue(u.ID, u.Email, u.Role)
	if err != nil {
		return "", nil, err
	}
	out := u
	out.PasswordHash = ""
	return tok, &out, nil
}

// RequestPasswordReset creates a reset token for the user when the email exists.
// Returns an empty rawToken (and nil error) when the email is unknown — anti-enumeration.
func (a *AuthService) RequestPasswordReset(ctx context.Context, email string) (rawToken string, err error) {
	if a == nil || a.Users == nil || a.Resets == nil {
		return "", fmt.Errorf("auth service not configured")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return "", nil
	}
	user, err := a.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	raw, hash, err := identity.NewOpaqueToken()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	id := ""
	if a.NewResetID != nil {
		id = a.NewResetID()
	}
	if id == "" {
		return "", fmt.Errorf("new reset id not configured")
	}
	if err := a.Resets.DeleteUnusedForUser(ctx, user.ID); err != nil {
		return "", err
	}
	if err := a.Resets.Create(ctx, identity.PasswordResetToken{
		ID:        id,
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: now.Add(identity.DefaultPasswordResetTTL),
		CreatedAt: now,
	}); err != nil {
		return "", err
	}
	return raw, nil
}

// ConfirmPasswordReset consumes a reset token, sets the new password, and issues a JWT.
func (a *AuthService) ConfirmPasswordReset(ctx context.Context, rawToken, password string) (token string, user *identity.User, err error) {
	if a == nil || a.Users == nil || a.Resets == nil || a.Passwords == nil || a.Tokens == nil {
		return "", nil, fmt.Errorf("auth service not configured")
	}
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" || len(password) < 8 {
		return "", nil, kernel.ErrInvalidRequest
	}
	stored, err := a.Resets.GetByTokenHash(ctx, identity.HashOpaqueToken(rawToken))
	if err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return "", nil, identity.ErrInvalidResetToken
		}
		return "", nil, err
	}
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	if stored.UsedAt != nil || !stored.ExpiresAt.After(now) {
		return "", nil, identity.ErrInvalidResetToken
	}
	hash, err := a.Passwords.Hash(password)
	if err != nil {
		return "", nil, err
	}
	if err := a.Users.UpdatePassword(ctx, stored.UserID, hash); err != nil {
		return "", nil, err
	}
	if err := a.Resets.Consume(ctx, stored.ID, now); err != nil {
		return "", nil, err
	}
	u, err := a.Users.GetByID(ctx, stored.UserID)
	if err != nil {
		return "", nil, err
	}
	tok, err := a.Tokens.Issue(u.ID, u.Email, u.Role)
	if err != nil {
		return "", nil, err
	}
	out := *u
	out.PasswordHash = ""
	return tok, &out, nil
}

// LoginWithPassword authenticates a local user and returns a session JWT.
func (a *AuthService) LoginWithPassword(ctx context.Context, email, password string) (string, error) {
	if a == nil || a.Users == nil || a.Passwords == nil || a.Tokens == nil {
		return "", fmt.Errorf("auth service not configured")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", identity.ErrInvalidCredentials
	}
	user, err := a.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return "", identity.ErrInvalidCredentials
		}
		return "", err
	}
	if user.PasswordHash == "" || !a.Passwords.Check(user.PasswordHash, password) {
		return "", identity.ErrInvalidCredentials
	}
	return a.Tokens.Issue(user.ID, user.Email, user.Role)
}

// BeginSSO starts an OAuth/OIDC/SAML login and returns the IdP redirect URL.
func (a *AuthService) BeginSSO(ctx context.Context, providerID, returnTo string) (authURL string, err error) {
	if a == nil || !a.SSOEnabled {
		return "", identity.ErrSSODisabled
	}
	p, ok := a.Providers[providerID]
	if !ok {
		return "", identity.ErrUnknownProvider
	}
	if a.States == nil {
		return "", fmt.Errorf("sso state store not configured")
	}
	if a.PublicBaseURL == "" {
		return "", fmt.Errorf("auth public_base_url not configured")
	}
	state, err := randomState()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	redirectURI := ssoCallbackURL(a.PublicBaseURL, providerID)
	var requestID string
	if starter, ok := p.(identity.AuthStarter); ok {
		authURL, requestID, err = starter.AuthURLWithID(state, redirectURI)
	} else {
		authURL, err = p.AuthURL(state, redirectURI)
	}
	if err != nil {
		return "", err
	}
	if err := a.States.Put(ctx, state, identity.SSOState{
		Provider:  providerID,
		ReturnTo:  sanitizeReturnTo(returnTo),
		RequestID: requestID,
		ExpiresAt: now.Add(10 * time.Minute),
	}); err != nil {
		return "", err
	}
	return authURL, nil
}

// CompleteSSOResult is returned after a successful federated login.
type CompleteSSOResult struct {
	Token    string
	ReturnTo string
}

// CompleteSSO finishes OAuth/OIDC/SAML login, JIT-provisions the user, and issues a JWT.
func (a *AuthService) CompleteSSO(ctx context.Context, providerID, code, state string) (*CompleteSSOResult, error) {
	if a == nil || !a.SSOEnabled {
		return nil, identity.ErrSSODisabled
	}
	p, ok := a.Providers[providerID]
	if !ok {
		return nil, identity.ErrUnknownProvider
	}
	if code == "" || state == "" {
		return nil, kernel.ErrInvalidRequest
	}
	st, err := a.States.Take(ctx, state)
	if err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil, identity.ErrInvalidSSOState
		}
		return nil, err
	}
	if st.Provider != providerID {
		return nil, identity.ErrInvalidSSOState
	}
	redirectURI := ssoCallbackURL(a.PublicBaseURL, providerID)
	var claims identity.FederatedClaims
	if exchanger, ok := p.(identity.AssertionExchanger); ok {
		ids := make([]string, 0, 2)
		if st.RequestID != "" {
			ids = append(ids, st.RequestID)
		}
		ids = append(ids, "")
		claims, err = exchanger.ExchangeAssertion(ctx, code, redirectURI, ids)
	} else {
		claims, err = p.Exchange(ctx, code, redirectURI)
	}
	if err != nil {
		return nil, err
	}
	claims.Provider = providerID

	user, err := identity.ResolveOrProvisionUser(ctx, identity.ResolveDeps{
		Users:      a.Users,
		Identities: a.Identities,
		NewUserID:  a.NewUserID,
		NewLinkID:  a.NewLinkID,
		Now:        a.Now,
	}, claims)
	if err != nil {
		return nil, err
	}
	tok, err := a.Tokens.Issue(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}
	return &CompleteSSOResult{Token: tok, ReturnTo: st.ReturnTo}, nil
}

// SSOMetadataXML returns SP metadata for a SAML provider, or ErrUnknownProvider / not supported.
func (a *AuthService) SSOMetadataXML(providerID string) ([]byte, error) {
	if a == nil || !a.SSOEnabled {
		return nil, identity.ErrSSODisabled
	}
	p, ok := a.Providers[providerID]
	if !ok {
		return nil, identity.ErrUnknownProvider
	}
	meta, ok := p.(identity.ServiceProviderMeta)
	if !ok {
		return nil, kernel.ErrInvalidRequest
	}
	return meta.MetadataXML()
}

// AppSSOCallbackURL builds the web UI URL that receives the session token.
func (a *AuthService) AppSSOCallbackURL(token, returnTo, errMsg string) string {
	base := strings.TrimRight(a.AppBaseURL, "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	u, err := url.Parse(base + "/auth/sso/callback")
	if err != nil {
		return base + "/auth/sso/callback"
	}
	q := u.Query()
	if errMsg != "" {
		q.Set("error", errMsg)
	} else {
		q.Set("token", token)
		if returnTo != "" {
			q.Set("redirect", returnTo)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func ssoCallbackURL(publicBaseURL, providerID string) string {
	base := strings.TrimRight(publicBaseURL, "/")
	return base + "/api/v1/platform/auth/sso/" + url.PathEscape(providerID) + "/callback"
}

func sanitizeReturnTo(returnTo string) string {
	returnTo = strings.TrimSpace(returnTo)
	if returnTo == "" {
		return ""
	}
	// Only allow relative in-app paths.
	if !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
		return ""
	}
	// Reject backslashes: some browsers normalize /\evil.com to //evil.com (open redirect).
	if strings.Contains(returnTo, `\`) {
		return ""
	}
	return returnTo
}

func randomState() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
