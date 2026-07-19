package tenancy

import (
	"context"
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

type memTeams struct {
	teams   map[string]Team
	members map[string]map[string]string // teamID -> userID -> role
	users   map[string]struct{ name, email string }
}

func newMemTeams() *memTeams {
	return &memTeams{
		teams: map[string]Team{
			"team_a": {ID: "team_a", TeamID: "team_a", OrganizationID: "org_x", Name: "Alpha"},
		},
		members: map[string]map[string]string{
			"team_a": {"user_owner": TeamRoleOwner},
		},
		users: map[string]struct{ name, email string }{
			"user_owner":  {name: "Owner", email: "owner@afi.local"},
			"user_member": {name: "Member", email: "member@afi.local"},
			"user_other":  {name: "Other", email: "other@afi.local"},
		},
	}
}

func (m *memTeams) ListByOrg(context.Context, string) ([]Team, error) { return nil, nil }
func (m *memTeams) ListByOrgForUser(context.Context, string, string) ([]Team, error) {
	return nil, nil
}
func (m *memTeams) Get(_ context.Context, teamID string) (*Team, error) {
	t, ok := m.teams[teamID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	return &t, nil
}
func (m *memTeams) OrgID(_ context.Context, teamID string) (string, error) {
	t, ok := m.teams[teamID]
	if !ok {
		return "", kernel.ErrNotFound
	}
	return t.OrganizationID, nil
}
func (m *memTeams) ListMembers(context.Context, string) ([]TeamMember, error) { return nil, nil }
func (m *memTeams) GetMember(_ context.Context, teamID, userID string) (*TeamMember, error) {
	role, ok := m.members[teamID][userID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	u := m.users[userID]
	return &TeamMember{UserID: userID, Name: u.name, Email: u.email, Role: role}, nil
}
func (m *memTeams) GetMemberRole(_ context.Context, teamID, userID string) (string, error) {
	role, ok := m.members[teamID][userID]
	if !ok {
		return "", kernel.ErrNotFound
	}
	return role, nil
}
func (m *memTeams) AddMember(_ context.Context, teamID, userID, role string) error {
	if m.members[teamID] == nil {
		m.members[teamID] = map[string]string{}
	}
	m.members[teamID][userID] = role
	return nil
}
func (m *memTeams) RemoveMember(_ context.Context, teamID, userID string) error {
	if _, ok := m.members[teamID][userID]; !ok {
		return kernel.ErrNotFound
	}
	delete(m.members[teamID], userID)
	return nil
}
func (m *memTeams) CountOwners(_ context.Context, teamID string) (int, error) {
	n := 0
	for _, role := range m.members[teamID] {
		if role == TeamRoleOwner {
			n++
		}
	}
	return n, nil
}
func (m *memTeams) CreateWithOwner(context.Context, Team, string) error { return nil }

type memOrgs struct {
	roles map[string]string // userID|orgID -> role
}

func (o *memOrgs) Count(context.Context) (int64, error)                         { return 0, nil }
func (o *memOrgs) ListForUser(context.Context, string) ([]Organization, error)  { return nil, nil }
func (o *memOrgs) CreateWithOwner(context.Context, Organization, string) error  { return nil }
func (o *memOrgs) ListMembers(context.Context, string) ([]OrgMember, error)     { return nil, nil }
func (o *memOrgs) AddMember(context.Context, string, string, string) error      { return nil }
func (o *memOrgs) CountOwners(context.Context, string) (int, error)             { return 0, nil }
func (o *memOrgs) ApplyRoleChange(context.Context, string, string, string, string, bool) error {
	return nil
}
func (o *memOrgs) GetMember(context.Context, string, string) (*OrgMember, error) {
	return nil, kernel.ErrNotFound
}
func (o *memOrgs) GetMemberRole(_ context.Context, userID, orgID string) (string, error) {
	role, ok := o.roles[userID+"|"+orgID]
	if !ok {
		return "", kernel.ErrNotFound
	}
	return role, nil
}

func TestAddTeamMemberRequiresOrgMembership(t *testing.T) {
	t.Parallel()
	teams := newMemTeams()
	orgs := &memOrgs{roles: map[string]string{"user_owner|org_x": OrgRoleOwner}}
	_, err := AddTeamMember(context.Background(), teams, orgs, "team_a", "user_other")
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestAddTeamMemberSuccess(t *testing.T) {
	t.Parallel()
	teams := newMemTeams()
	orgs := &memOrgs{roles: map[string]string{
		"user_owner|org_x":  OrgRoleOwner,
		"user_member|org_x": OrgRoleMember,
	}}
	m, err := AddTeamMember(context.Background(), teams, orgs, "team_a", "user_member")
	if err != nil {
		t.Fatal(err)
	}
	if m.Role != TeamRoleMember || m.UserID != "user_member" {
		t.Fatalf("%+v", m)
	}
}

func TestRemoveSoleTeamOwner(t *testing.T) {
	t.Parallel()
	teams := newMemTeams()
	err := RemoveTeamMember(context.Background(), teams, "team_a", "user_owner")
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestCanManageTeam(t *testing.T) {
	t.Parallel()
	teams := newMemTeams()
	orgs := &memOrgs{roles: map[string]string{
		"user_owner|org_x":  OrgRoleMember,
		"user_admin|org_x":  OrgRoleAdmin,
		"user_member|org_x": OrgRoleMember,
	}}
	ok, err := CanManageTeam(context.Background(), teams, orgs, "team_a", "user_owner")
	if err != nil || !ok {
		t.Fatalf("owner manage: ok=%v err=%v", ok, err)
	}
	ok, err = CanManageTeam(context.Background(), teams, orgs, "team_a", "user_admin")
	if err != nil || !ok {
		t.Fatalf("admin manage: ok=%v err=%v", ok, err)
	}
	ok, err = CanManageTeam(context.Background(), teams, orgs, "team_a", "user_member")
	if err != nil || ok {
		t.Fatalf("member manage: ok=%v err=%v", ok, err)
	}
}

func TestCanAccessTeam(t *testing.T) {
	t.Parallel()
	teams := newMemTeams()
	orgs := &memOrgs{roles: map[string]string{
		"user_owner|org_x":  OrgRoleMember,
		"user_admin|org_x":  OrgRoleAdmin,
		"user_member|org_x": OrgRoleMember,
	}}
	ok, err := CanAccessTeam(context.Background(), teams, orgs, "team_a", "user_owner")
	if err != nil || !ok {
		t.Fatalf("member access: ok=%v err=%v", ok, err)
	}
	ok, err = CanAccessTeam(context.Background(), teams, orgs, "team_a", "user_admin")
	if err != nil || !ok {
		t.Fatalf("admin access: ok=%v err=%v", ok, err)
	}
	ok, err = CanAccessTeam(context.Background(), teams, orgs, "team_a", "user_member")
	if err != nil || ok {
		t.Fatalf("non-member access: ok=%v err=%v", ok, err)
	}
}
