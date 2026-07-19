package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

type teamMembersFake struct {
	fakePlatform
	teams   map[string]*Team
	members map[string][]TeamMember // teamID -> members
	listed  []string                // userIDs that called ListTeams
}

func newTeamMembersFake() *teamMembersFake {
	f := &teamMembersFake{
		teams: map[string]*Team{
			"team_a": {ID: "team_a", TeamID: "team_a", OrganizationID: "org_x", Name: "Alpha"},
			"team_b": {ID: "team_b", TeamID: "team_b", OrganizationID: "org_x", Name: "Beta"},
		},
		members: map[string][]TeamMember{
			"team_a": {
				{UserID: "user_owner", Name: "Owner", Email: "owner@afi.local", Role: TeamRoleOwner},
				{UserID: "user_tm", Name: "TM", Email: "tm@afi.local", Role: "member"},
			},
			"team_b": {
				{UserID: "user_admin", Name: "Admin", Email: "admin@afi.local", Role: TeamRoleOwner},
			},
		},
	}
	f.allowed = map[string]bool{
		"user_admin|org_x":  true,
		"user_owner|org_x":  true,
		"user_tm|org_x":     true,
		"user_member|org_x": true,
	}
	f.admins = map[string]bool{
		"user_admin|org_x": true,
	}
	f.teamOrg = map[string]string{
		"team_a": "org_x",
		"team_b": "org_x",
	}
	f.teamAccess = map[string]bool{
		"user_admin|team_a": true,
		"user_admin|team_b": true,
		"user_owner|team_a": true,
		"user_tm|team_a":    true,
	}
	f.teamManage = map[string]bool{
		"user_admin|team_a": true,
		"user_admin|team_b": true,
		"user_owner|team_a": true,
	}
	f.teamRoleChange = map[string]bool{
		"user_admin|team_a": true,
		"user_admin|team_b": true,
		"user_owner|team_a": true,
	}
	return f
}

func (f *teamMembersFake) ListTeams(_ context.Context, orgID, userID string) ([]Team, error) {
	f.listed = append(f.listed, userID)
	admin := f.admins[userID+"|"+orgID]
	var out []Team
	for _, t := range f.teams {
		if t.OrganizationID != orgID {
			continue
		}
		if admin || f.teamAccess[userID+"|"+t.ID] {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (f *teamMembersFake) GetTeam(_ context.Context, teamID string) (*Team, error) {
	t, ok := f.teams[teamID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *teamMembersFake) ListTeamMembers(_ context.Context, teamID string) ([]TeamMember, error) {
	return f.members[teamID], nil
}

func (f *teamMembersFake) AddTeamMember(_ context.Context, teamID, userID string) (*TeamMember, error) {
	if !f.allowed[userID+"|org_x"] {
		return nil, kernel.ErrInvalidRequest
	}
	for _, m := range f.members[teamID] {
		if m.UserID == userID {
			return nil, kernel.ErrInvalidRequest
		}
	}
	m := TeamMember{UserID: userID, Name: userID, Email: userID + "@afi.local", Role: "member"}
	f.members[teamID] = append(f.members[teamID], m)
	return &m, nil
}

func (f *teamMembersFake) RemoveTeamMember(_ context.Context, teamID, userID string) error {
	list := f.members[teamID]
	idx := -1
	var role string
	owners := 0
	for i, m := range list {
		if m.Role == TeamRoleOwner {
			owners++
		}
		if m.UserID == userID {
			idx = i
			role = m.Role
		}
	}
	if idx < 0 {
		return kernel.ErrNotFound
	}
	if role == TeamRoleOwner && owners <= 1 {
		return kernel.ErrInvalidRequest
	}
	f.members[teamID] = append(list[:idx], list[idx+1:]...)
	return nil
}

func (f *teamMembersFake) UpdateTeamMemberRole(_ context.Context, teamID, actorUserID, targetUserID, role string) (*TeamMember, error) {
	ok, err := f.CanChangeTeamRoles(context.Background(), teamID, actorUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, kernel.ErrUnauthorized
	}
	switch role {
	case TeamRoleOwner, TeamRoleAdmin, TeamRoleMember:
	default:
		return nil, kernel.ErrInvalidRequest
	}
	list := f.members[teamID]
	owners := 0
	idx := -1
	for i, m := range list {
		if m.Role == TeamRoleOwner {
			owners++
		}
		if m.UserID == targetUserID {
			idx = i
		}
	}
	if idx < 0 {
		return nil, kernel.ErrNotFound
	}
	if list[idx].Role == TeamRoleOwner && role != TeamRoleOwner && owners <= 1 {
		return nil, kernel.ErrInvalidRequest
	}
	list[idx].Role = role
	f.members[teamID] = list
	cp := list[idx]
	return &cp, nil
}

func teamServer(api *teamMembersFake) *Server {
	return &Server{
		cfg: testCfg(), api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default(),
	}
}

func TestListTeamsFiltersForMember(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_owner", "owner@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/organizations/org_x/teams", nil)
	req.SetPathValue("orgID", "org_x")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var teams []Team
	if err := json.Unmarshal(rr.Body.Bytes(), &teams); err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].ID != "team_a" {
		t.Fatalf("teams=%+v", teams)
	}
}

func TestListTeamsAdminSeesAll(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/organizations/org_x/teams", nil)
	req.SetPathValue("orgID", "org_x")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var teams []Team
	if err := json.Unmarshal(rr.Body.Bytes(), &teams); err != nil {
		t.Fatal(err)
	}
	if len(teams) != 2 {
		t.Fatalf("teams=%+v", teams)
	}
}

func TestGetTeamForbiddenForNonMember(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_member", "member@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/teams/team_a", nil)
	req.SetPathValue("teamID", "team_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAddTeamMemberAsOwner(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_owner", "owner@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/teams/team_a/members",
		bytes.NewBufferString(`{"user_id":"user_member"}`))
	req.SetPathValue("teamID", "team_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAddTeamMemberForbiddenForPlainMember(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_tm", "tm@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/teams/team_a/members",
		bytes.NewBufferString(`{"user_id":"user_member"}`))
	req.SetPathValue("teamID", "team_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRemoveSoleTeamOwnerRejected(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/platform/teams/team_a/members/user_owner", nil)
	req.SetPathValue("teamID", "team_a")
	req.SetPathValue("userID", "user_owner")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRemoveTeamMemberSuccess(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_owner", "owner@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/platform/teams/team_a/members/user_tm", nil)
	req.SetPathValue("teamID", "team_a")
	req.SetPathValue("userID", "user_tm")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateTeamMemberRoleAsOrgAdmin(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/teams/team_a/members/user_tm",
		bytes.NewBufferString(`{"role":"admin"}`))
	req.SetPathValue("teamID", "team_a")
	req.SetPathValue("userID", "user_tm")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var member TeamMember
	if err := json.Unmarshal(rr.Body.Bytes(), &member); err != nil {
		t.Fatal(err)
	}
	if member.Role != TeamRoleAdmin {
		t.Fatalf("role=%s", member.Role)
	}
}

func TestUpdateTeamMemberRoleForbiddenForPlainMember(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_tm", "tm@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/teams/team_a/members/user_tm",
		bytes.NewBufferString(`{"role":"admin"}`))
	req.SetPathValue("teamID", "team_a")
	req.SetPathValue("userID", "user_tm")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateTeamMemberRoleForbiddenForTeamAdmin(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	api.members["team_a"] = append(api.members["team_a"], TeamMember{
		UserID: "user_tadmin", Name: "TAdmin", Email: "tadmin@afi.local", Role: TeamRoleAdmin,
	})
	api.allowed["user_tadmin|org_x"] = true
	api.teamAccess["user_tadmin|team_a"] = true
	api.teamManage["user_tadmin|team_a"] = true
	// team admin can manage members but must not change roles
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_tadmin", "tadmin@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/teams/team_a/members/user_tadmin",
		bytes.NewBufferString(`{"role":"owner"}`))
	req.SetPathValue("teamID", "team_a")
	req.SetPathValue("userID", "user_tadmin")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateTeamMemberRoleAsTeamOwner(t *testing.T) {
	t.Parallel()
	api := newTeamMembersFake()
	s := teamServer(api)
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_owner", "owner@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/teams/team_a/members/user_tm",
		bytes.NewBufferString(`{"role":"admin"}`))
	req.SetPathValue("teamID", "team_a")
	req.SetPathValue("userID", "user_tm")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
