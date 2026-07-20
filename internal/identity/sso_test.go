package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

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
	if _, ok := m.byEmail[user.Email]; ok {
		return kernel.ErrConflict
	}
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
	key := item.Provider + "|" + item.Subject
	if _, ok := m.byKey[key]; ok {
		return kernel.ErrConflict
	}
	cp := item
	m.byKey[key] = &cp
	return nil
}

func TestResolveOrProvisionUser_JITCreate(t *testing.T) {
	t.Parallel()
	users := newMemUsers()
	idents := newMemIdentities()
	seq := 0
	deps := identity.ResolveDeps{
		Users:      users,
		Identities: idents,
		NewUserID: func() string {
			seq++
			return "user_new"
		},
		NewLinkID: func() string { return "link_1" },
		Now:       func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
	user, err := identity.ResolveOrProvisionUser(context.Background(), deps, identity.FederatedClaims{
		Provider: "google", Issuer: "https://accounts.google.com", Subject: "sub-1",
		Email: "new@example.com", EmailVerified: true, Name: "New User",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "user_new" || user.Email != "new@example.com" || user.Role != "member" || user.PasswordHash != "" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if _, err := idents.GetByProviderSubject(context.Background(), "google", "sub-1"); err != nil {
		t.Fatal(err)
	}
}

func TestResolveOrProvisionUser_LinkExistingByEmail(t *testing.T) {
	t.Parallel()
	users := newMemUsers()
	idents := newMemIdentities()
	existing := identity.User{
		ID: "user_1", Email: "a@example.com", Name: "A", Role: "admin",
		PasswordHash: "hash", CreatedAt: time.Now().UTC(),
	}
	_ = users.Create(context.Background(), existing)
	deps := identity.ResolveDeps{
		Users: users, Identities: idents,
		NewUserID: func() string { return "should-not" },
		NewLinkID: func() string { return "link_2" },
	}
	user, err := identity.ResolveOrProvisionUser(context.Background(), deps, identity.FederatedClaims{
		Provider: "okta", Subject: "okta-sub", Email: "A@Example.com", EmailVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "user_1" {
		t.Fatalf("want existing user, got %s", user.ID)
	}
	link, err := idents.GetByProviderSubject(context.Background(), "okta", "okta-sub")
	if err != nil {
		t.Fatal(err)
	}
	if link.UserID != "user_1" {
		t.Fatalf("link user %s", link.UserID)
	}
}

func TestResolveOrProvisionUser_ExistingSubject(t *testing.T) {
	t.Parallel()
	users := newMemUsers()
	idents := newMemIdentities()
	_ = users.Create(context.Background(), identity.User{ID: "user_1", Email: "a@example.com", Name: "A", Role: "member"})
	_ = idents.Create(context.Background(), identity.ExternalIdentity{
		ID: "l1", UserID: "user_1", Provider: "google", Subject: "sub-1", Email: "a@example.com",
	})
	deps := identity.ResolveDeps{
		Users: users, Identities: idents,
		NewUserID: func() string { return "x" },
		NewLinkID: func() string { return "y" },
	}
	user, err := identity.ResolveOrProvisionUser(context.Background(), deps, identity.FederatedClaims{
		Provider: "google", Subject: "sub-1", Email: "other@example.com", EmailVerified: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "user_1" {
		t.Fatalf("got %s", user.ID)
	}
}

func TestResolveOrProvisionUser_RejectUnverified(t *testing.T) {
	t.Parallel()
	deps := identity.ResolveDeps{
		Users: newMemUsers(), Identities: newMemIdentities(),
		NewUserID: func() string { return "u" },
		NewLinkID: func() string { return "l" },
	}
	_, err := identity.ResolveOrProvisionUser(context.Background(), deps, identity.FederatedClaims{
		Provider: "google", Subject: "sub", Email: "a@b.c", EmailVerified: false,
	})
	if !errors.Is(err, identity.ErrEmailNotVerified) {
		t.Fatalf("got %v", err)
	}
}
