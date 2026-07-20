package oauthoidc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/adapters/oauthoidc"
)

func TestAuthURLUsesDiscovery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "http://example/auth",
			"token_endpoint":         "http://example/token",
			"userinfo_endpoint":      "http://example/userinfo",
			"issuer":                 "http://issuer",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	p, err := oauthoidc.New(oauthoidc.Config{
		ID: "test", Type: "oidc", Issuer: srv.URL,
		ClientID: "cid", ClientSecret: "secret",
		RequireEmailVerified: true,
		HTTPClient:           srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	url, err := p.AuthURL("state123", "http://localhost/callback")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "state123") || !strings.Contains(url, "cid") {
		t.Fatalf("auth url=%s", url)
	}
}

func TestSAMLRejected(t *testing.T) {
	t.Parallel()
	_, err := oauthoidc.New(oauthoidc.Config{
		ID: "saml1", Type: "saml", ClientID: "x", ClientSecret: "y",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRedirectURI(t *testing.T) {
	t.Parallel()
	got := oauthoidc.RedirectURI("http://localhost:8081/", "google")
	want := "http://localhost:8081/api/v1/platform/auth/sso/google/callback"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
