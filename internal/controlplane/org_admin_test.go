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

type orgAdminFake struct {
	fakePlatform
	orgs    map[string]*Organization
	members map[string][]OrgMember
	users   map[string]*User
}

func newOrgAdminFake() *orgAdminFake {
	return &orgAdminFake{
		orgs:    map[string]*Organization{},
		members: map[string][]OrgMember{},
		users: map[string]*User{
			"admin@afi.local": {ID: "user_admin", Email: "admin@afi.local", Name: "Admin"},
			"other@afi.local": {ID: "user_other", Email: "other@afi.local", Name: "Other"},
		},
	}
}

func (f *orgAdminFake) GetUserByEmail(_ context.Context, email string) (*User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	return u, nil
}

func (f *orgAdminFake) CreateOrganization(_ context.Context, name, creatorUserID string) (*Organization, error) {
	o := &Organization{ID: "org_new", Name: name, CreatedAt: time.Now().UTC()}
	f.orgs[o.ID] = o
	f.members[o.ID] = []OrgMember{{UserID: creatorUserID, Email: "admin@afi.local", Name: "Admin"}}
	return o, nil
}

func (f *orgAdminFake) ListOrgMembers(_ context.Context, orgID string) ([]OrgMember, error) {
	return f.members[orgID], nil
}

func (f *orgAdminFake) AddOrgMemberByEmail(_ context.Context, orgID, email string) (*OrgMember, error) {
	u, err := f.GetUserByEmail(context.Background(), email)
	if err != nil {
		return nil, err
	}
	m := OrgMember{UserID: u.ID, Email: u.Email, Name: u.Name, Role: OrgRoleMember}
	f.members[orgID] = append(f.members[orgID], m)
	return &m, nil
}

func (f *orgAdminFake) memberRole(orgID, userID string) (string, bool) {
	for _, m := range f.members[orgID] {
		if m.UserID == userID {
			return m.Role, true
		}
	}
	return "", false
}

func (f *orgAdminFake) UpdateOrgMemberRole(_ context.Context, orgID, actorUserID, targetUserID, role string) (*OrgMember, error) {
	switch role {
	case OrgRoleOwner, OrgRoleAdmin, OrgRoleMember:
	default:
		return nil, kernel.ErrInvalidRequest
	}
	actorRole, ok := f.memberRole(orgID, actorUserID)
	if !ok {
		return nil, kernel.ErrNotFound
	}
	if actorRole != OrgRoleOwner {
		return nil, kernel.ErrUnauthorized
	}
	owners := 0
	for _, m := range f.members[orgID] {
		if m.Role == OrgRoleOwner {
			owners++
		}
	}
	targetRole, ok := f.memberRole(orgID, targetUserID)
	if !ok {
		return nil, kernel.ErrNotFound
	}
	if targetRole == OrgRoleOwner && role != OrgRoleOwner && owners <= 1 {
		return nil, kernel.ErrInvalidRequest
	}
	var updated *OrgMember
	for i := range f.members[orgID] {
		m := &f.members[orgID][i]
		if role == OrgRoleOwner && m.UserID == actorUserID && actorUserID != targetUserID {
			m.Role = OrgRoleAdmin
		}
		if m.UserID == targetUserID {
			m.Role = role
			cp := *m
			updated = &cp
		}
	}
	return updated, nil
}

func TestCreateOrganization(t *testing.T) {
	t.Parallel()
	api := newOrgAdminFake()
	s := &Server{
		cfg: testCfg(), api: api, members: &fakeMembers{}, publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations", bytes.NewBufferString(`{"name":"Acme"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var org Organization
	if err := json.Unmarshal(rr.Body.Bytes(), &org); err != nil {
		t.Fatal(err)
	}
	if org.Name != "Acme" || org.ID == "" {
		t.Fatalf("%+v", org)
	}
}

func TestAddOrgMemberSuccess(t *testing.T) {
	t.Parallel()
	api := newOrgAdminFake()
	api.orgs["org_x"] = &Organization{ID: "org_x", Name: "X"}
	api.members["org_x"] = []OrgMember{{UserID: "user_admin", Email: "admin@afi.local"}}
	s := &Server{
		cfg: testCfg(), api: api,
		members:   &fakeMembers{allowed: map[string]bool{"user_admin|org_x": true}},
		publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_x/members",
		bytes.NewBufferString(`{"email":"other@afi.local"}`))
	req.SetPathValue("orgID", "org_x")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAddOrgMemberUnknownEmail(t *testing.T) {
	t.Parallel()
	api := newOrgAdminFake()
	api.orgs["org_x"] = &Organization{ID: "org_x", Name: "X"}
	api.members["org_x"] = []OrgMember{{UserID: "user_admin", Email: "admin@afi.local"}}
	s := &Server{
		cfg: testCfg(), api: api,
		members:   &fakeMembers{allowed: map[string]bool{"user_admin|org_x": true}},
		publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_x/members",
		bytes.NewBufferString(`{"email":"missing@afi.local"}`))
	req.SetPathValue("orgID", "org_x")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAddOrgMemberForbiddenForNonMember(t *testing.T) {
	t.Parallel()
	api := newOrgAdminFake()
	api.orgs["org_x"] = &Organization{ID: "org_x", Name: "X"}
	s := &Server{
		cfg: testCfg(), api: api,
		members:   &fakeMembers{allowed: map[string]bool{}},
		publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_x/members",
		bytes.NewBufferString(`{"email":"other@afi.local"}`))
	req.SetPathValue("orgID", "org_x")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateOrgMemberRoleSuccess(t *testing.T) {
	t.Parallel()
	api := newOrgAdminFake()
	api.orgs["org_x"] = &Organization{ID: "org_x", Name: "X"}
	api.members["org_x"] = []OrgMember{
		{UserID: "user_admin", Email: "admin@afi.local", Name: "Admin", Role: OrgRoleOwner},
		{UserID: "user_other", Email: "other@afi.local", Name: "Other", Role: OrgRoleMember},
	}
	s := &Server{
		cfg: testCfg(), api: api,
		members:   &fakeMembers{allowed: map[string]bool{"user_admin|org_x": true}},
		publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/organizations/org_x/members/user_other",
		bytes.NewBufferString(`{"role":"admin"}`))
	req.SetPathValue("orgID", "org_x")
	req.SetPathValue("userID", "user_other")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var m OrgMember
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.Role != OrgRoleAdmin {
		t.Fatalf("role=%s", m.Role)
	}
}

func TestUpdateOrgMemberRoleForbiddenForNonOwner(t *testing.T) {
	t.Parallel()
	api := newOrgAdminFake()
	api.orgs["org_x"] = &Organization{ID: "org_x", Name: "X"}
	api.members["org_x"] = []OrgMember{
		{UserID: "user_admin", Email: "admin@afi.local", Name: "Admin", Role: OrgRoleOwner},
		{UserID: "user_other", Email: "other@afi.local", Name: "Other", Role: OrgRoleAdmin},
	}
	s := &Server{
		cfg: testCfg(), api: api,
		members:   &fakeMembers{allowed: map[string]bool{"user_other|org_x": true}},
		publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_other", "other@afi.local", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/organizations/org_x/members/user_admin",
		bytes.NewBufferString(`{"role":"member"}`))
	req.SetPathValue("orgID", "org_x")
	req.SetPathValue("userID", "user_admin")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateOrgMemberRoleRejectsSoleOwnerDemotion(t *testing.T) {
	t.Parallel()
	api := newOrgAdminFake()
	api.orgs["org_x"] = &Organization{ID: "org_x", Name: "X"}
	api.members["org_x"] = []OrgMember{
		{UserID: "user_admin", Email: "admin@afi.local", Name: "Admin", Role: OrgRoleOwner},
	}
	s := &Server{
		cfg: testCfg(), api: api,
		members:   &fakeMembers{allowed: map[string]bool{"user_admin|org_x": true}},
		publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/organizations/org_x/members/user_admin",
		bytes.NewBufferString(`{"role":"admin"}`))
	req.SetPathValue("orgID", "org_x")
	req.SetPathValue("userID", "user_admin")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
