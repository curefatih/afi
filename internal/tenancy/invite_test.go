package tenancy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

type memInvites struct {
	byID    map[string]*OrgInvite
	byHash  map[string]string // hash -> inviteID
	orgName string
}

func newMemInvites() *memInvites {
	return &memInvites{
		byID:    map[string]*OrgInvite{},
		byHash:  map[string]string{},
		orgName: "Acme",
	}
}

func (m *memInvites) Get(_ context.Context, inviteID string) (*OrgInvite, error) {
	inv, ok := m.byID[inviteID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *inv
	return &cp, nil
}

func (m *memInvites) GetPendingByOrgEmail(_ context.Context, orgID, email string) (*OrgInvite, error) {
	for _, inv := range m.byID {
		if inv.OrganizationID == orgID && inv.Email == email && inv.Status == InviteStatusPending {
			cp := *inv
			return &cp, nil
		}
	}
	return nil, kernel.ErrNotFound
}

func (m *memInvites) GetByTokenHash(_ context.Context, tokenHash string) (*OrgInvite, string, error) {
	id, ok := m.byHash[tokenHash]
	if !ok {
		return nil, "", kernel.ErrNotFound
	}
	inv := m.byID[id]
	cp := *inv
	return &cp, m.orgName, nil
}

func (m *memInvites) ListByOrg(context.Context, string) ([]OrgInvite, error) { return nil, nil }

func (m *memInvites) Insert(_ context.Context, inv OrgInvite, tokenHash string) error {
	cp := inv
	m.byID[inv.ID] = &cp
	m.byHash[tokenHash] = inv.ID
	return nil
}

func (m *memInvites) UpdateToken(_ context.Context, inviteID, tokenHash string, expiresAt time.Time) error {
	inv, ok := m.byID[inviteID]
	if !ok {
		return kernel.ErrNotFound
	}
	for h, id := range m.byHash {
		if id == inviteID {
			delete(m.byHash, h)
		}
	}
	m.byHash[tokenHash] = inviteID
	inv.ExpiresAt = expiresAt
	return nil
}

func (m *memInvites) MarkAccepted(_ context.Context, inviteID string, at time.Time) error {
	inv, ok := m.byID[inviteID]
	if !ok {
		return kernel.ErrNotFound
	}
	inv.Status = InviteStatusAccepted
	inv.AcceptedAt = &at
	return nil
}

func (m *memInvites) MarkRevoked(_ context.Context, inviteID string) error {
	inv, ok := m.byID[inviteID]
	if !ok {
		return kernel.ErrNotFound
	}
	inv.Status = InviteStatusRevoked
	return nil
}

type memLookup struct {
	users map[string]struct{ id, name, email string }
}

func (l *memLookup) FindByEmail(_ context.Context, email string) (string, string, string, error) {
	u, ok := l.users[NormalizeEmail(email)]
	if !ok {
		return "", "", "", kernel.ErrNotFound
	}
	return u.id, u.name, u.email, nil
}

type memCreator struct {
	lookup *memLookup
}

func (c *memCreator) CreateUser(_ context.Context, id, email, name, passwordHash string) (*identity.User, error) {
	if passwordHash == "" {
		return nil, kernel.ErrInvalidRequest
	}
	c.lookup.users[NormalizeEmail(email)] = struct{ id, name, email string }{id, name, email}
	return &identity.User{ID: id, Email: email, Name: name, Role: "member"}, nil
}

func TestInviteOrgMemberCreatesPendingInvite(t *testing.T) {
	t.Parallel()
	orgs := &memOrgs{roles: map[string]string{"user_admin|org_x": OrgRoleOwner}}
	invites := newMemInvites()
	users := &memLookup{users: map[string]struct{ id, name, email string }{}}
	out, raw, err := InviteOrgMember(context.Background(), orgs, invites, users, "inv_1", "org_x", "new@afi.local", "user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "invited" || raw == "" || out.Invite == nil {
		t.Fatalf("%+v raw=%q", out, raw)
	}
}

func TestInviteOrgMemberAddsExistingUser(t *testing.T) {
	t.Parallel()
	orgs := &memOrgs{roles: map[string]string{"user_admin|org_x": OrgRoleOwner}}
	invites := newMemInvites()
	users := &memLookup{users: map[string]struct{ id, name, email string }{
		"other@afi.local": {"user_other", "Other", "other@afi.local"},
	}}
	out, raw, err := InviteOrgMember(context.Background(), orgs, invites, users, "inv_1", "org_x", "other@afi.local", "user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "added" || raw != "" || out.Member == nil {
		t.Fatalf("%+v", out)
	}
}

func TestAcceptInviteCreatesUser(t *testing.T) {
	t.Parallel()
	orgs := &memOrgs{roles: map[string]string{}}
	invites := newMemInvites()
	users := &memLookup{users: map[string]struct{ id, name, email string }{}}
	out, raw, err := InviteOrgMember(context.Background(), orgs, invites, users, "inv_1", "org_x", "new@afi.local", "user_admin")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	creator := &memCreator{lookup: users}
	member, user, err := AcceptInvite(
		context.Background(), orgs, invites, users, creator, raw,
		AcceptInviteInput{Name: "New", PasswordHash: "hash"},
		"user_new",
	)
	if err != nil {
		t.Fatal(err)
	}
	if member.UserID != "user_new" || user.Email != "new@afi.local" {
		t.Fatalf("member=%+v user=%+v", member, user)
	}
}

func TestRevokeInvite(t *testing.T) {
	t.Parallel()
	orgs := &memOrgs{roles: map[string]string{}}
	invites := newMemInvites()
	users := &memLookup{users: map[string]struct{ id, name, email string }{}}
	out, _, err := InviteOrgMember(context.Background(), orgs, invites, users, "inv_1", "org_x", "new@afi.local", "user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := RevokeInvite(context.Background(), invites, "org_x", out.Invite.ID); err != nil {
		t.Fatal(err)
	}
	if err := RevokeInvite(context.Background(), invites, "org_x", out.Invite.ID); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
