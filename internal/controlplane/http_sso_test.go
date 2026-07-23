package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/auth"
	"github.com/curefatih/afi/internal/adapters/memory"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

type ssoMemUsers struct {
	byID    map[string]*identity.User
	byEmail map[string]*identity.User
}

func (m *ssoMemUsers) GetByEmail(_ context.Context, email string) (*identity.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *u
	return &cp, nil
}
func (m *ssoMemUsers) GetByID(_ context.Context, id string) (*identity.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *u
	return &cp, nil
}
func (m *ssoMemUsers) Create(_ context.Context, user identity.User) error {
	cp := user
	if m.byID == nil {
		m.byID = map[string]*identity.User{}
		m.byEmail = map[string]*identity.User{}
	}
	m.byID[user.ID] = &cp
	m.byEmail[user.Email] = &cp
	return nil
}

type ssoMemIdentities struct {
	byKey map[string]*identity.ExternalIdentity
}

func (m *ssoMemIdentities) GetByProviderSubject(_ context.Context, provider, subject string) (*identity.ExternalIdentity, error) {
	item, ok := m.byKey[provider+"|"+subject]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *item
	return &cp, nil
}
func (m *ssoMemIdentities) Create(_ context.Context, item identity.ExternalIdentity) error {
	if m.byKey == nil {
		m.byKey = map[string]*identity.ExternalIdentity{}
	}
	cp := item
	m.byKey[item.Provider+"|"+item.Subject] = &cp
	return nil
}

type ssoFakeFed struct{}

func (ssoFakeFed) Name() string        { return "google" }
func (ssoFakeFed) DisplayName() string { return "Google" }
func (ssoFakeFed) Type() string        { return "oidc" }
func (ssoFakeFed) AuthURL(state, redirectURI string) (string, error) {
	return "https://idp.example/authorize?state=" + state + "&redirect_uri=" + redirectURI, nil
}
func (ssoFakeFed) Exchange(context.Context, string, string) (identity.FederatedClaims, error) {
	return identity.FederatedClaims{
		Provider: "google", Subject: "sub-http", Email: "sso-http@example.com",
		EmailVerified: true, Name: "SSO HTTP",
	}, nil
}

type ssoFakeSAML struct{}

func (ssoFakeSAML) Name() string        { return "okta" }
func (ssoFakeSAML) DisplayName() string { return "Okta SAML" }
func (ssoFakeSAML) Type() string        { return "saml" }
func (ssoFakeSAML) AuthURL(state, redirectURI string) (string, error) {
	u, _, err := ssoFakeSAML{}.AuthURLWithID(state, redirectURI)
	return u, err
}
func (ssoFakeSAML) AuthURLWithID(state, redirectURI string) (string, string, error) {
	return "https://idp.example/sso?RelayState=" + state + "&acs=" + redirectURI, "id-req-1", nil
}
func (ssoFakeSAML) Exchange(context.Context, string, string) (identity.FederatedClaims, error) {
	return identity.FederatedClaims{}, fmt.Errorf("use ExchangeAssertion")
}
func (ssoFakeSAML) ExchangeAssertion(_ context.Context, response, _ string, ids []string) (identity.FederatedClaims, error) {
	if response != "saml-b64" {
		return identity.FederatedClaims{}, fmt.Errorf("bad response")
	}
	if len(ids) == 0 || ids[0] != "id-req-1" {
		return identity.FederatedClaims{}, fmt.Errorf("bad request ids %v", ids)
	}
	return identity.FederatedClaims{
		Provider: "okta", Subject: "saml-sub", Email: "saml@example.com",
		EmailVerified: true, Name: "SAML User",
	}, nil
}
func (ssoFakeSAML) MetadataXML() ([]byte, error) {
	return []byte(`<EntityDescriptor entityID="sp"/>`), nil
}

func TestSSOStartAndCallback(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Auth.JWTSecret = "secret"
	cfg.Auth.TokenTTL = time.Hour
	cfg.Auth.PublicBaseURL = "http://cp.test"
	cfg.Mail.PublicAppURL = "http://app.test"
	cfg.Auth.SSO.Enabled = true

	users := &ssoMemUsers{}
	idents := &ssoMemIdentities{}
	tokens := auth.NewService(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	states := memory.NewSSOStateStore(time.Minute)
	authSvc := &platform.AuthService{
		Users: users, Identities: idents, Tokens: tokens, Passwords: tokens,
		States: states, Providers: map[string]identity.FederationProvider{"google": ssoFakeFed{}},
		SSOEnabled: true, PublicBaseURL: cfg.Auth.PublicBaseURL, AppBaseURL: cfg.Mail.PublicAppURL,
		NewUserID: func() string { return "user_sso" },
		NewLinkID: func() string { return "ext_sso" },
	}
	s := &Server{cfg: cfg, auth: authSvc, log: slog.Default()}
	h := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/auth/sso/providers", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "google") {
		t.Fatalf("providers: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/platform/auth/sso/google/start?redirect=/app/dashboard", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("start status %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "https://idp.example/authorize") {
		t.Fatalf("location %s", loc)
	}

	_ = states.Put(context.Background(), "cbstate", identity.SSOState{
		Provider: "google", ReturnTo: "/app/dashboard", ExpiresAt: time.Now().Add(time.Minute),
	})
	req = httptest.NewRequest(http.MethodGet, "/api/v1/platform/auth/sso/google/callback?code=abc&state=cbstate", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("callback status %d body=%s", rec.Code, rec.Body.String())
	}
	appLoc := rec.Header().Get("Location")
	if !strings.Contains(appLoc, "http://app.test/auth/sso/callback") || !strings.Contains(appLoc, "token=") {
		t.Fatalf("app redirect %s", appLoc)
	}
	u, err := users.GetByEmail(context.Background(), "sso-http@example.com")
	if err != nil || u.ID != "user_sso" {
		t.Fatalf("user=%+v err=%v", u, err)
	}
}

func TestSAMLACSAndMetadata(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Auth.JWTSecret = "secret"
	cfg.Auth.TokenTTL = time.Hour
	cfg.Auth.PublicBaseURL = "http://cp.test"
	cfg.Mail.PublicAppURL = "http://app.test"
	cfg.Auth.SSO.Enabled = true

	users := &ssoMemUsers{}
	idents := &ssoMemIdentities{}
	tokens := auth.NewService(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	states := memory.NewSSOStateStore(time.Minute)
	authSvc := &platform.AuthService{
		Users: users, Identities: idents, Tokens: tokens, Passwords: tokens,
		States: states, Providers: map[string]identity.FederationProvider{"okta": ssoFakeSAML{}},
		SSOEnabled: true, PublicBaseURL: cfg.Auth.PublicBaseURL, AppBaseURL: cfg.Mail.PublicAppURL,
		NewUserID: func() string { return "user_saml" },
		NewLinkID: func() string { return "ext_saml" },
	}
	s := &Server{cfg: cfg, auth: authSvc, log: slog.Default()}
	h := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/auth/sso/okta/metadata", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "EntityDescriptor") {
		t.Fatalf("metadata: %d %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "samlmetadata") {
		t.Fatalf("content-type %s", ct)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/platform/auth/sso/okta/start", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("start %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	relay := u.Query().Get("RelayState")
	if relay == "" {
		t.Fatalf("missing RelayState in %s", loc)
	}

	form := url.Values{}
	form.Set("SAMLResponse", "saml-b64")
	form.Set("RelayState", relay)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/sso/okta/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("acs %d body=%s", rec.Code, rec.Body.String())
	}
	appLoc := rec.Header().Get("Location")
	if !strings.Contains(appLoc, "token=") {
		t.Fatalf("app redirect %s", appLoc)
	}

	// Bad relay state
	form.Set("RelayState", "nope")
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/sso/okta/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "error=") {
		t.Fatalf("bad relay: %d %s", rec.Code, rec.Header().Get("Location"))
	}
}
