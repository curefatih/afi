package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/auth"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/mail"
)

type signupMemResets struct {
	byHash map[string]*identity.PasswordResetToken
	byID   map[string]*identity.PasswordResetToken
}

func (m *signupMemResets) Create(_ context.Context, token identity.PasswordResetToken) error {
	if m.byHash == nil {
		m.byHash = map[string]*identity.PasswordResetToken{}
		m.byID = map[string]*identity.PasswordResetToken{}
	}
	cp := token
	m.byHash[token.TokenHash] = &cp
	m.byID[token.ID] = &cp
	return nil
}

func (m *signupMemResets) DeleteUnusedForUser(_ context.Context, userID string) error {
	for hash, t := range m.byHash {
		if t.UserID == userID && t.UsedAt == nil {
			delete(m.byHash, hash)
			delete(m.byID, t.ID)
		}
	}
	return nil
}

func (m *signupMemResets) GetByTokenHash(_ context.Context, hash string) (*identity.PasswordResetToken, error) {
	t, ok := m.byHash[hash]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (m *signupMemResets) Consume(_ context.Context, id string, usedAt time.Time) error {
	t, ok := m.byID[id]
	if !ok {
		return kernel.ErrNotFound
	}
	cp := usedAt
	t.UsedAt = &cp
	return nil
}

func TestAuthFeaturesAndRegisterGated(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Auth.JWTSecret = "secret"
	cfg.Auth.TokenTTL = time.Hour
	cfg.Mail.PublicAppURL = "http://app.test"
	cfg.Mail.DefaultProvider = mail.ProviderLog

	users := &ssoMemUsers{}
	tokens := auth.NewService(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	authSvc := &platform.AuthService{
		Users: users, Tokens: tokens, Passwords: tokens,
		SignupEnabled: false, NewUserID: func() string { return "user_reg" },
	}
	s := &Server{cfg: cfg, auth: authSvc, log: slog.Default()}
	h := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/auth/features", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"signup_enabled":false`) {
		t.Fatalf("features: %d %s", rec.Code, rec.Body.String())
	}

	body, _ := json.Marshal(map[string]string{
		"email": "new@example.com", "name": "New", "password": "password1",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/register", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("register disabled: %d %s", rec.Code, rec.Body.String())
	}

	authSvc.SignupEnabled = true
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/register", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/register", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPasswordResetHTTPRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Auth.JWTSecret = "secret"
	cfg.Auth.TokenTTL = time.Hour
	cfg.Mail.PublicAppURL = "http://app.test"
	cfg.Mail.DefaultProvider = mail.ProviderLog

	hash, err := auth.HashPassword("oldpass12")
	if err != nil {
		t.Fatal(err)
	}
	users := &ssoMemUsers{}
	_ = users.Create(context.Background(), identity.User{
		ID: "u1", Email: "reset@afi.local", Name: "R", Role: "member", PasswordHash: hash,
	})
	resets := &signupMemResets{}
	tokens := auth.NewService(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	authSvc := &platform.AuthService{
		Users: users, Resets: resets, Tokens: tokens, Passwords: tokens,
		NewResetID: func() string { return "pwreset_1" },
	}
	s := &Server{cfg: cfg, auth: authSvc, log: slog.Default()}
	h := s.Handler()

	body, _ := json.Marshal(map[string]string{"email": "reset@afi.local"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/password-reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("request reset: %d %s", rec.Code, rec.Body.String())
	}
	// Capture raw token via service (HTTP path emails it; LogSender in tests).
	raw, err := authSvc.RequestPasswordReset(context.Background(), "reset@afi.local")
	if err != nil || raw == "" {
		t.Fatalf("raw=%q err=%v", raw, err)
	}

	confirmBody, _ := json.Marshal(map[string]string{"password": "newpass12"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/password-reset/"+raw, bytes.NewReader(confirmBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}

	loginBody, _ := json.Marshal(map[string]string{"email": "reset@afi.local", "password": "newpass12"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/login", bytes.NewReader(loginBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login after reset: %d %s", rec.Code, rec.Body.String())
	}

	// Unknown email still 200
	unk, _ := json.Marshal(map[string]string{"email": "nobody@afi.local"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/auth/password-reset", bytes.NewReader(unk))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unknown email: %d %s", rec.Code, rec.Body.String())
	}
}
