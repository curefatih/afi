package oauthoidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// Config describes a platform-wide OAuth2 or OIDC IdP.
type Config struct {
	ID                   string
	Type                 string // oidc | oauth2 (saml reserved for a future adapter)
	DisplayName          string
	Issuer               string
	ClientID             string
	ClientSecret         string
	Scopes               []string
	AuthURL              string // required for oauth2 without discovery
	TokenURL             string
	UserInfoURL          string
	RequireEmailVerified bool
	HTTPClient           *http.Client
}

// Provider implements identity.FederationProvider for OAuth2 and OIDC.
type Provider struct {
	cfg Config

	mu         sync.Mutex
	discovered bool
	authURL    string
	tokenURL   string
	userInfo   string
}

func New(cfg Config) (*Provider, error) {
	cfg.ID = strings.TrimSpace(cfg.ID)
	cfg.Type = strings.ToLower(strings.TrimSpace(cfg.Type))
	if cfg.Type == "" {
		if cfg.Issuer != "" {
			cfg.Type = "oidc"
		} else {
			cfg.Type = "oauth2"
		}
	}
	if cfg.Type == "saml" {
		return nil, fmt.Errorf("sso provider %q: saml is not implemented yet; use oidc or oauth2", cfg.ID)
	}
	if cfg.Type != "oidc" && cfg.Type != "oauth2" {
		return nil, fmt.Errorf("sso provider %q: unsupported type %q", cfg.ID, cfg.Type)
	}
	if cfg.ID == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("sso provider: id, client_id, and client_secret are required")
	}
	if cfg.DisplayName == "" {
		cfg.DisplayName = cfg.ID
	}
	if len(cfg.Scopes) == 0 {
		if cfg.Type == "oidc" {
			cfg.Scopes = []string{"openid", "email", "profile"}
		} else {
			cfg.Scopes = []string{"email", "profile"}
		}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	p := &Provider{cfg: cfg}
	p.authURL = cfg.AuthURL
	p.tokenURL = cfg.TokenURL
	p.userInfo = cfg.UserInfoURL
	return p, nil
}

func (p *Provider) Name() string        { return p.cfg.ID }
func (p *Provider) DisplayName() string { return p.cfg.DisplayName }
func (p *Provider) Type() string        { return p.cfg.Type }

func (p *Provider) AuthURL(state, redirectURI string) (string, error) {
	if err := p.ensureEndpoints(context.Background()); err != nil {
		return "", err
	}
	cfg := p.oauth2Config(redirectURI)
	return cfg.AuthCodeURL(state), nil
}

func (p *Provider) Exchange(ctx context.Context, code, redirectURI string) (identity.FederatedClaims, error) {
	if err := p.ensureEndpoints(ctx); err != nil {
		return identity.FederatedClaims{}, err
	}
	// oauth2.Config.Exchange reads the client from context; without this it uses
	// http.DefaultClient and ignores timeouts/proxy/TLS on p.cfg.HTTPClient.
	if p.cfg.HTTPClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, p.cfg.HTTPClient)
	}
	tok, err := p.oauth2Config(redirectURI).Exchange(ctx, code)
	if err != nil {
		return identity.FederatedClaims{}, fmt.Errorf("token exchange: %w", err)
	}

	claims := identity.FederatedClaims{Provider: p.cfg.ID, Issuer: p.cfg.Issuer}

	if raw, ok := tok.Extra("id_token").(string); ok && raw != "" {
		parsed, err := parseIDTokenClaims(raw)
		if err != nil {
			return identity.FederatedClaims{}, err
		}
		if err := validateIDTokenClaims(parsed, p.cfg.Issuer, p.cfg.ClientID); err != nil {
			return identity.FederatedClaims{}, err
		}
		claims.Subject = parsed.Subject
		claims.Email = parsed.Email
		claims.EmailVerified = parsed.EmailVerified
		claims.Name = parsed.Name
		if parsed.Issuer != "" {
			claims.Issuer = parsed.Issuer
		}
	}

	if claims.Subject == "" || claims.Email == "" {
		info, err := p.fetchUserInfo(ctx, tok.AccessToken)
		if err != nil {
			return identity.FederatedClaims{}, err
		}
		if claims.Subject == "" {
			claims.Subject = firstNonEmpty(info.Sub, info.ID)
		}
		if claims.Email == "" {
			claims.Email = info.Email
		}
		if claims.Name == "" {
			claims.Name = firstNonEmpty(info.Name, info.PreferredUsername)
		}
		if !claims.EmailVerified {
			claims.EmailVerified = info.EmailVerified
		}
	}

	// Plain OAuth2 providers often omit email_verified; when not required, treat a
	// present email as verified after a successful authenticated token exchange.
	if !p.cfg.RequireEmailVerified && claims.Email != "" {
		claims.EmailVerified = true
	}

	claims.Provider = p.cfg.ID
	return claims, nil
}

func (p *Provider) oauth2Config(redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       p.cfg.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.authURL,
			TokenURL: p.tokenURL,
		},
	}
}

func (p *Provider) ensureEndpoints(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.authURL != "" && p.tokenURL != "" {
		return nil
	}
	if p.cfg.Issuer == "" {
		return fmt.Errorf("sso provider %q: issuer or auth_url/token_url required", p.cfg.ID)
	}
	disc, err := p.discover(ctx, p.cfg.Issuer)
	if err != nil {
		return err
	}
	p.authURL = disc.AuthorizationEndpoint
	p.tokenURL = disc.TokenEndpoint
	if p.userInfo == "" {
		p.userInfo = disc.UserinfoEndpoint
	}
	p.discovered = true
	return nil
}

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	Issuer                string `json:"issuer"`
}

func (p *Provider) discover(ctx context.Context, issuer string) (oidcDiscovery, error) {
	issuer = strings.TrimRight(issuer, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return oidcDiscovery{}, err
	}
	resp, err := p.cfg.HTTPClient.Do(req)
	if err != nil {
		return oidcDiscovery{}, fmt.Errorf("oidc discovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return oidcDiscovery{}, fmt.Errorf("oidc discovery: status %d: %s", resp.StatusCode, string(body))
	}
	var disc oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return oidcDiscovery{}, fmt.Errorf("oidc discovery decode: %w", err)
	}
	if disc.AuthorizationEndpoint == "" || disc.TokenEndpoint == "" {
		return oidcDiscovery{}, fmt.Errorf("oidc discovery: missing endpoints")
	}
	return disc, nil
}

type idTokenClaims struct {
	Issuer        string `json:"iss"`
	Subject       string `json:"sub"`
	Audience      any    `json:"aud"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	ExpiresAt     int64  `json:"exp"`
}

// parseIDTokenClaims decodes the JWT payload without signature verification.
// The token is obtained via a confidential client_secret exchange over TLS; claim
// checks (iss/aud/exp) still apply. JWKS signature verification can be added later.
func parseIDTokenClaims(raw string) (idTokenClaims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return idTokenClaims{}, fmt.Errorf("invalid id_token format")
	}
	payload, err := jwt.NewParser().DecodeSegment(parts[1])
	if err != nil {
		return idTokenClaims{}, fmt.Errorf("invalid id_token payload: %w", err)
	}
	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return idTokenClaims{}, fmt.Errorf("invalid id_token claims: %w", err)
	}
	return claims, nil
}

func validateIDTokenClaims(claims idTokenClaims, issuer, clientID string) error {
	if issuer != "" && claims.Issuer != "" {
		want := strings.TrimRight(issuer, "/")
		got := strings.TrimRight(claims.Issuer, "/")
		if got != want {
			return fmt.Errorf("id_token iss mismatch")
		}
	}
	if clientID != "" && claims.Audience != nil && !audienceContains(claims.Audience, clientID) {
		return fmt.Errorf("id_token aud mismatch")
	}
	if claims.ExpiresAt > 0 && time.Unix(claims.ExpiresAt, 0).Before(time.Now().Add(-30*time.Second)) {
		return fmt.Errorf("id_token expired")
	}
	return nil
}

type userInfo struct {
	Sub               string `json:"sub"`
	ID                string `json:"id"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

func (p *Provider) fetchUserInfo(ctx context.Context, accessToken string) (userInfo, error) {
	if p.userInfo == "" {
		return userInfo{}, fmt.Errorf("sso provider %q: no userinfo endpoint and incomplete id_token", p.cfg.ID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfo, nil)
	if err != nil {
		return userInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := p.cfg.HTTPClient.Do(req)
	if err != nil {
		return userInfo{}, fmt.Errorf("userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return userInfo{}, fmt.Errorf("userinfo status %d: %s", resp.StatusCode, string(body))
	}
	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return userInfo{}, fmt.Errorf("userinfo decode: %w", err)
	}
	return info, nil
}

func audienceContains(aud any, clientID string) bool {
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []any:
		for _, a := range v {
			if s, ok := a.(string); ok && s == clientID {
				return true
			}
		}
	case []string:
		for _, a := range v {
			if a == clientID {
				return true
			}
		}
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// RedirectURI builds the control-plane callback URL for a provider.
func RedirectURI(publicBaseURL, providerID string) string {
	base := strings.TrimRight(publicBaseURL, "/")
	return base + "/api/v1/platform/auth/sso/" + url.PathEscape(providerID) + "/callback"
}

var _ identity.FederationProvider = (*Provider)(nil)
