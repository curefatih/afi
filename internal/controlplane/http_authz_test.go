package controlplane

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

type fakeMembers struct {
	allowed map[string]bool
}

func (f *fakeMembers) IsOrgMember(_ context.Context, userID, orgID string) (bool, error) {
	return f.allowed[userID+"|"+orgID], nil
}

func (f *fakeMembers) GetTeamOrgID(_ context.Context, teamID string) (string, error) {
	if teamID == "team_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetProjectOrgID(_ context.Context, projectID string) (string, error) {
	if projectID == "proj_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetProviderOrgID(_ context.Context, providerID string) (string, error) {
	if providerID == "prov_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetRouteOrgID(_ context.Context, routeID string) (string, error) {
	if routeID == "route_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

type fakePublisher struct {
	err   error
	calls int
}

func (p *fakePublisher) PublishSnapshot(context.Context) error {
	p.calls++
	return p.err
}

type fakePlatform struct {
	fakeMembers
	created *Project
}

func (f *fakePlatform) GetUserByEmail(context.Context, string) (*User, error) {
	return nil, kernel.ErrNotFound
}
func (f *fakePlatform) GetUserByID(context.Context, string) (*User, error) {
	return nil, kernel.ErrNotFound
}
func (f *fakePlatform) ListOrganizationsForUser(context.Context, string) ([]Organization, error) {
	return nil, nil
}
func (f *fakePlatform) ListTeams(context.Context, string) ([]Team, error) { return nil, nil }
func (f *fakePlatform) GetTeam(context.Context, string) (*Team, error) {
	return nil, kernel.ErrNotFound
}
func (f *fakePlatform) ListTeamMembers(context.Context, string) ([]TeamMember, error) {
	return nil, nil
}
func (f *fakePlatform) ListProjects(context.Context, string) ([]Project, error) { return nil, nil }
func (f *fakePlatform) CreateProject(_ context.Context, orgID, teamID, name string) (*Project, error) {
	f.created = &Project{ID: "proj_new", OrganizationID: orgID, TeamID: teamID, Name: name}
	return f.created, nil
}
func (f *fakePlatform) ListAPIKeys(context.Context, string) ([]APIKey, error) { return nil, nil }
func (f *fakePlatform) CreateAPIKey(context.Context, string, string, string, string) (*APIKey, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) ListProviders(context.Context, string) ([]Provider, error) { return nil, nil }
func (f *fakePlatform) CreateProvider(context.Context, string, string, string, string, string) (*Provider, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateProvider(context.Context, string, string, string, string) (*Provider, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteProvider(context.Context, string) error { return nil }
func (f *fakePlatform) ListRoutes(context.Context, string) ([]Route, error) { return nil, nil }
func (f *fakePlatform) CreateRoute(context.Context, string, string, string, string) (*Route, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateRoute(context.Context, string, string, string, string) (*Route, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteRoute(context.Context, string) error { return nil }
func (f *fakePlatform) ListUsage(context.Context, string, int) ([]UsageEvent, error) { return nil, nil }

func testCfg() *kernel.Config {
	cfg := &kernel.Config{}
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Auth.TokenTTL = time.Hour
	cfg.Auth.InternalToken = "internal-secret"
	return cfg
}

func TestInternalSeedRequiresToken(t *testing.T) {
	t.Parallel()
	s := &Server{
		cfg:       testCfg(),
		publisher: &fakePublisher{},
		log:       slog.Default(),
		api:       &fakePlatform{},
		members:   &fakeMembers{},
	}
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/seed", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401 body=%s", rr.Code, rr.Body.String())
	}
}

func TestInternalPublishRequiresToken(t *testing.T) {
	t.Parallel()
	s := &Server{
		cfg:       testCfg(),
		publisher: &fakePublisher{},
		log:       slog.Default(),
		api:       &fakePlatform{},
		members:   &fakeMembers{},
	}
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/snapshots/publish", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401 body=%s", rr.Code, rr.Body.String())
	}
}

func TestInternalPublishAcceptsToken(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{}
	s := &Server{
		cfg:       testCfg(),
		publisher: pub,
		log:       slog.Default(),
	}
	h := s.requireInternal(func(w http.ResponseWriter, r *http.Request) {
		if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/snapshots/publish", nil)
	req.Header.Set("X-AFI-Internal-Token", "internal-secret")
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if pub.calls != 1 {
		t.Fatalf("publish calls=%d", pub.calls)
	}
}

func TestOrgMemberForbidden(t *testing.T) {
	t.Parallel()
	s := &Server{
		cfg:     testCfg(),
		members: &fakeMembers{allowed: map[string]bool{}},
		log:     slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_1", "a@b.c", "admin")
	if err != nil {
		t.Fatal(err)
	}
	h := s.requireAuth(s.requireOrgMemberFromPath("orgID", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/organizations/org_a/teams", nil)
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}

func TestOrgMemberAllowed(t *testing.T) {
	t.Parallel()
	s := &Server{
		cfg:     testCfg(),
		members: &fakeMembers{allowed: map[string]bool{"user_1|org_a": true}},
		log:     slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_1", "a@b.c", "admin")
	if err != nil {
		t.Fatal(err)
	}
	h := s.requireAuth(s.requireOrgMemberFromPath("orgID", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateProjectSurfacesPublishError(t *testing.T) {
	t.Parallel()
	pub := &fakePublisher{err: errors.New("publish boom")}
	api := &fakePlatform{fakeMembers: fakeMembers{allowed: map[string]bool{"user_1|org_a": true}}}
	s := &Server{
		cfg:       testCfg(),
		members:   api,
		publisher: pub,
		api:       api,
		log:       slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_1", "a@b.c", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/projects",
		bytes.NewBufferString(`{"name":"P"}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleCreateProject))(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want 500 body=%s", rr.Code, rr.Body.String())
	}
	if pub.calls != 1 {
		t.Fatalf("expected publish call")
	}
}
