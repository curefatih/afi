package platform_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/auth"
	"github.com/curefatih/afi/internal/adapters/memory"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

type memUsers struct {
	byID    map[string]*identity.User
	byEmail map[string]*identity.User
}

func newMemUsers() *memUsers {
	return &memUsers{byID: map[string]*identity.User{}, byEmail: map[string]*identity.User{}}
}

func (m *memUsers) GetByEmail(_ context.Context, email string) (*identity.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *u
	return &cp, nil
}
func (m *memUsers) GetByID(_ context.Context, id string) (*identity.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *u
	return &cp, nil
}
func (m *memUsers) Create(_ context.Context, user identity.User) error {
	cp := user
	m.byID[user.ID] = &cp
	m.byEmail[user.Email] = &cp
	return nil
}
func (m *memUsers) UpdatePassword(_ context.Context, userID, passwordHash string) error {
	u, ok := m.byID[userID]
	if !ok {
		return kernel.ErrNotFound
	}
	u.PasswordHash = passwordHash
	return nil
}

type memResets struct {
	byHash map[string]*identity.PasswordResetToken
	byID   map[string]*identity.PasswordResetToken
}

func newMemResets() *memResets {
	return &memResets{
		byHash: map[string]*identity.PasswordResetToken{},
		byID:   map[string]*identity.PasswordResetToken{},
	}
}

func (m *memResets) Create(_ context.Context, token identity.PasswordResetToken) error {
	cp := token
	m.byHash[token.TokenHash] = &cp
	m.byID[token.ID] = &cp
	return nil
}

func (m *memResets) DeleteUnusedForUser(_ context.Context, userID string) error {
	for hash, t := range m.byHash {
		if t.UserID == userID && t.UsedAt == nil {
			delete(m.byHash, hash)
			delete(m.byID, t.ID)
		}
	}
	return nil
}

func (m *memResets) GetByTokenHash(_ context.Context, hash string) (*identity.PasswordResetToken, error) {
	t, ok := m.byHash[hash]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (m *memResets) Consume(_ context.Context, id string, usedAt time.Time) error {
	t, ok := m.byID[id]
	if !ok {
		return kernel.ErrNotFound
	}
	cp := usedAt
	t.UsedAt = &cp
	return nil
}

type memIdentities struct {
	byKey map[string]*identity.ExternalIdentity
}

func newMemIdentities() *memIdentities {
	return &memIdentities{byKey: map[string]*identity.ExternalIdentity{}}
}
func (m *memIdentities) GetByProviderSubject(_ context.Context, provider, subject string) (*identity.ExternalIdentity, error) {
	item, ok := m.byKey[provider+"|"+subject]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *item
	return &cp, nil
}
func (m *memIdentities) Create(_ context.Context, item identity.ExternalIdentity) error {
	cp := item
	m.byKey[item.Provider+"|"+item.Subject] = &cp
	return nil
}

type fakeFed struct {
	id     string
	url    string
	claims identity.FederatedClaims
}

func (f *fakeFed) Name() string        { return f.id }
func (f *fakeFed) DisplayName() string { return f.id }
func (f *fakeFed) Type() string        { return "oidc" }
func (f *fakeFed) AuthURL(state, _ string) (string, error) {
	return f.url + "?state=" + state, nil
}
func (f *fakeFed) Exchange(context.Context, string, string) (identity.FederatedClaims, error) {
	return f.claims, nil
}

func TestLoginWithPassword_RejectsEmptyHash(t *testing.T) {
	t.Parallel()
	users := newMemUsers()
	_ = users.Create(context.Background(), identity.User{
		ID: "u1", Email: "sso@afi.local", Name: "SSO", Role: "member", PasswordHash: "",
	})
	svc := &platform.AuthService{
		Users:     users,
		Passwords: auth.NewService("secret", time.Hour),
		Tokens:    auth.NewService("secret", time.Hour),
	}
	_, err := svc.LoginWithPassword(context.Background(), "sso@afi.local", "anything")
	if !errors.Is(err, identity.ErrInvalidCredentials) {
		t.Fatalf("got %v", err)
	}
}

func TestLoginWithPassword_Success(t *testing.T) {
	t.Parallel()
	hash, err := auth.HashPassword("admin")
	if err != nil {
		t.Fatal(err)
	}
	users := newMemUsers()
	_ = users.Create(context.Background(), identity.User{
		ID: "u1", Email: "admin@afi.local", Name: "Admin", Role: "admin", PasswordHash: hash,
	})
	tokens := auth.NewService("secret", time.Hour)
	svc := &platform.AuthService{Users: users, Passwords: tokens, Tokens: tokens}
	tok, err := svc.LoginWithPassword(context.Background(), "admin@afi.local", "admin")
	if err != nil || tok == "" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
}

func TestCompleteSSO_JIT(t *testing.T) {
	t.Parallel()
	users := newMemUsers()
	idents := newMemIdentities()
	tokens := auth.NewService("secret", time.Hour)
	states := memory.NewSSOStateStore(time.Minute)
	fed := &fakeFed{
		id:  "google",
		url: "https://idp.example/auth",
		claims: identity.FederatedClaims{
			Provider: "google", Subject: "sub-9", Email: "jit@example.com",
			EmailVerified: true, Name: "JIT",
		},
	}
	svc := &platform.AuthService{
		Users: users, Identities: idents, Tokens: tokens, Passwords: tokens,
		States: states, Providers: map[string]identity.FederationProvider{"google": fed},
		SSOEnabled: true, PublicBaseURL: "http://localhost:8081", AppBaseURL: "http://localhost:3000",
		NewUserID: func() string { return "user_jit" },
		NewLinkID: func() string { return "ext_1" },
	}
	authURL, err := svc.BeginSSO(context.Background(), "google", "/app/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	if authURL == "" {
		t.Fatal("empty auth url")
	}
	// Extract state from stored map by completing with Take via Begin's Put — re-put known state.
	_ = states.Put(context.Background(), "fixedstate", identity.SSOState{Provider: "google", ReturnTo: "/app/dashboard", ExpiresAt: time.Now().Add(time.Minute)})
	result, err := svc.CompleteSSO(context.Background(), "google", "code", "fixedstate")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" || result.ReturnTo != "/app/dashboard" {
		t.Fatalf("%+v", result)
	}
	u, err := users.GetByEmail(context.Background(), "jit@example.com")
	if err != nil || u.ID != "user_jit" {
		t.Fatalf("user=%+v err=%v", u, err)
	}
}

func TestRegister_Disabled(t *testing.T) {
	t.Parallel()
	tokens := auth.NewService("secret", time.Hour)
	svc := &platform.AuthService{
		Users: newMemUsers(), Passwords: tokens, Tokens: tokens,
		SignupEnabled: false, NewUserID: func() string { return "u_new" },
	}
	_, _, err := svc.Register(context.Background(), "a@b.c", "A", "password1")
	if !errors.Is(err, identity.ErrSignupDisabled) {
		t.Fatalf("got %v", err)
	}
}

func TestRegister_SuccessAndConflict(t *testing.T) {
	t.Parallel()
	users := newMemUsers()
	tokens := auth.NewService("secret", time.Hour)
	svc := &platform.AuthService{
		Users: users, Passwords: tokens, Tokens: tokens,
		SignupEnabled: true, NewUserID: func() string { return "u_reg" },
	}
	tok, user, err := svc.Register(context.Background(), "New@Example.com", "New User", "password1")
	if err != nil || tok == "" || user == nil || user.Email != "new@example.com" || user.Role != "member" {
		t.Fatalf("tok=%q user=%+v err=%v", tok, user, err)
	}
	_, _, err = svc.Register(context.Background(), "new@example.com", "Dup", "password1")
	if !errors.Is(err, kernel.ErrConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestPasswordReset_RoundTrip(t *testing.T) {
	t.Parallel()
	hash, err := auth.HashPassword("oldpass12")
	if err != nil {
		t.Fatal(err)
	}
	users := newMemUsers()
	_ = users.Create(context.Background(), identity.User{
		ID: "u1", Email: "reset@afi.local", Name: "R", Role: "member", PasswordHash: hash,
	})
	resets := newMemResets()
	tokens := auth.NewService("secret", time.Hour)
	svc := &platform.AuthService{
		Users: users, Resets: resets, Passwords: tokens, Tokens: tokens,
		NewResetID: func() string { return "reset_1" },
	}
	raw, err := svc.RequestPasswordReset(context.Background(), "reset@afi.local")
	if err != nil || raw == "" {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
	unknown, err := svc.RequestPasswordReset(context.Background(), "missing@afi.local")
	if err != nil || unknown != "" {
		t.Fatalf("unknown=%q err=%v", unknown, err)
	}
	tok, user, err := svc.ConfirmPasswordReset(context.Background(), raw, "newpass12")
	if err != nil || tok == "" || user == nil {
		t.Fatalf("tok=%q user=%+v err=%v", tok, user, err)
	}
	_, err = svc.LoginWithPassword(context.Background(), "reset@afi.local", "newpass12")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.ConfirmPasswordReset(context.Background(), raw, "another12")
	if !errors.Is(err, identity.ErrInvalidResetToken) {
		t.Fatalf("reuse got %v", err)
	}
}

func TestPasswordReset_Expired(t *testing.T) {
	t.Parallel()
	users := newMemUsers()
	_ = users.Create(context.Background(), identity.User{
		ID: "u1", Email: "exp@afi.local", Name: "E", Role: "member", PasswordHash: "x",
	})
	resets := newMemResets()
	tokens := auth.NewService("secret", time.Hour)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	svc := &platform.AuthService{
		Users: users, Resets: resets, Passwords: tokens, Tokens: tokens,
		NewResetID: func() string { return "reset_exp" },
		Now:        func() time.Time { return now },
	}
	raw, err := svc.RequestPasswordReset(context.Background(), "exp@afi.local")
	if err != nil || raw == "" {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
	svc.Now = func() time.Time { return now.Add(2 * time.Hour) }
	_, _, err = svc.ConfirmPasswordReset(context.Background(), raw, "newpass12")
	if !errors.Is(err, identity.ErrInvalidResetToken) {
		t.Fatalf("got %v", err)
	}
}
