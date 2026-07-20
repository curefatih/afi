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
	authURL, err := svc.BeginSSO("google", "/app/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	if authURL == "" {
		t.Fatal("empty auth url")
	}
	// Extract state from stored map by completing with Take via Begin's Put — re-put known state.
	_ = states.Put("fixedstate", identity.SSOState{Provider: "google", ReturnTo: "/app/dashboard", ExpiresAt: time.Now().Add(time.Minute)})
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
