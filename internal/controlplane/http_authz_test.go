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

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type fakeMembers struct {
	allowed        map[string]bool
	admins         map[string]bool
	owners         map[string]bool
	keyOrg         map[string]string
	teamOrg        map[string]string
	teamAccess     map[string]bool // userID|teamID
	teamManage     map[string]bool // userID|teamID
	teamRoleChange map[string]bool // userID|teamID — org admin / team owner only
}

func (f *fakeMembers) IsOrgMember(_ context.Context, userID, orgID string) (bool, error) {
	return f.allowed[userID+"|"+orgID], nil
}

func (f *fakeMembers) IsOrgAdmin(_ context.Context, userID, orgID string) (bool, error) {
	if f.admins != nil {
		return f.admins[userID+"|"+orgID], nil
	}
	return f.allowed[userID+"|"+orgID], nil
}

func (f *fakeMembers) IsOrgOwner(_ context.Context, userID, orgID string) (bool, error) {
	if f.owners != nil {
		return f.owners[userID+"|"+orgID], nil
	}
	return false, nil
}

func (f *fakeMembers) GetAPIKeyOrgID(_ context.Context, keyID string) (string, error) {
	if f.keyOrg != nil {
		if org, ok := f.keyOrg[keyID]; ok {
			return org, nil
		}
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetTeamOrgID(_ context.Context, teamID string) (string, error) {
	if f.teamOrg != nil {
		if org, ok := f.teamOrg[teamID]; ok {
			return org, nil
		}
	}
	if teamID == "team_ok" || teamID == "team_new" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) CanAccessTeam(ctx context.Context, teamID, userID string) (bool, error) {
	if f.teamAccess != nil {
		return f.teamAccess[userID+"|"+teamID], nil
	}
	orgID, err := f.GetTeamOrgID(ctx, teamID)
	if err != nil {
		return false, err
	}
	admin, err := f.IsOrgAdmin(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}
	return false, nil
}

func (f *fakeMembers) CanManageTeam(ctx context.Context, teamID, userID string) (bool, error) {
	if f.teamManage != nil {
		return f.teamManage[userID+"|"+teamID], nil
	}
	orgID, err := f.GetTeamOrgID(ctx, teamID)
	if err != nil {
		return false, err
	}
	return f.IsOrgAdmin(ctx, userID, orgID)
}

func (f *fakeMembers) CanChangeTeamRoles(ctx context.Context, teamID, userID string) (bool, error) {
	if f.teamRoleChange != nil {
		return f.teamRoleChange[userID+"|"+teamID], nil
	}
	orgID, err := f.GetTeamOrgID(ctx, teamID)
	if err != nil {
		return false, err
	}
	return f.IsOrgAdmin(ctx, userID, orgID)
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

func (f *fakeMembers) GetQuotaOrgID(_ context.Context, quotaID string) (string, error) {
	if quotaID == "quota_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetPolicyOrgID(_ context.Context, policyID string) (string, error) {
	if policyID == "pol_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetWasmHookOrgID(_ context.Context, hookID string) (string, error) {
	if hookID == "wasm_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetCredentialOrgID(_ context.Context, credentialID string) (string, error) {
	if credentialID == "cred_ok" {
		return "org_a", nil
	}
	return "", kernel.ErrNotFound
}

func (f *fakeMembers) GetCredentialAssignmentOrgID(_ context.Context, assignmentID string) (string, error) {
	if assignmentID == "casg_ok" {
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
func (f *fakePlatform) CreateOrganization(context.Context, string, string) (*Organization, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) ListOrgMembers(context.Context, string) ([]OrgMember, error) { return nil, nil }
func (f *fakePlatform) AddOrgMemberByEmail(context.Context, string, string) (*OrgMember, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) InviteOrgMember(context.Context, string, string, string) (*InviteOutcome, string, error) {
	return nil, "", errors.New("unused")
}
func (f *fakePlatform) ListOrgInvites(context.Context, string) ([]OrgInvite, error) { return nil, nil }
func (f *fakePlatform) RevokeOrgInvite(context.Context, string, string) error {
	return errors.New("unused")
}
func (f *fakePlatform) ResendOrgInvite(context.Context, string, string) (*OrgInvite, string, error) {
	return nil, "", errors.New("unused")
}
func (f *fakePlatform) PreviewOrgInvite(context.Context, string) (*InvitePreview, error) {
	return nil, kernel.ErrNotFound
}
func (f *fakePlatform) AcceptOrgInvite(context.Context, string, string, string) (*OrgMember, *User, error) {
	return nil, nil, errors.New("unused")
}
func (f *fakePlatform) GetOrganization(context.Context, string) (*Organization, error) {
	return nil, kernel.ErrNotFound
}
func (f *fakePlatform) SetOrgMailProvider(context.Context, string, string) (*Organization, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateOrgMemberRole(context.Context, string, string, string, string) (*OrgMember, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) ListOrganizationsForUser(context.Context, string) ([]Organization, error) {
	return nil, nil
}
func (f *fakePlatform) ListTeams(context.Context, string, string) ([]Team, error) { return nil, nil }
func (f *fakePlatform) CreateTeam(context.Context, string, string, string) (*Team, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) GetTeam(context.Context, string) (*Team, error) {
	return nil, kernel.ErrNotFound
}
func (f *fakePlatform) ListTeamMembers(context.Context, string) ([]TeamMember, error) {
	return nil, nil
}
func (f *fakePlatform) AddTeamMember(context.Context, string, string) (*TeamMember, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) RemoveTeamMember(context.Context, string, string) error {
	return errors.New("unused")
}
func (f *fakePlatform) UpdateTeamMemberRole(context.Context, string, string, string, string) (*TeamMember, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) ListProjects(context.Context, string, string) ([]Project, error) {
	return nil, nil
}
func (f *fakePlatform) CreateProject(_ context.Context, orgID, teamID, name string) (*Project, error) {
	f.created = &Project{ID: "proj_new", OrganizationID: orgID, TeamID: teamID, Name: name}
	return f.created, nil
}
func (f *fakePlatform) ListAPIKeys(context.Context, string) ([]APIKey, error) { return nil, nil }
func (f *fakePlatform) ListOrgAPIKeys(context.Context, string) ([]APIKey, error) {
	return nil, nil
}
func (f *fakePlatform) GetAPIKey(context.Context, string) (*APIKey, error) {
	return nil, kernel.ErrNotFound
}
func (f *fakePlatform) CreateAPIKey(context.Context, string, string, string, string, string, string) (*APIKey, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteAPIKey(context.Context, string) error                { return kernel.ErrNotFound }
func (f *fakePlatform) ListProviders(context.Context, string) ([]Provider, error) { return nil, nil }
func (f *fakePlatform) ListProviderHealth(context.Context, string, time.Time, time.Time) ([]ProviderHealth, error) {
	return nil, nil
}
func (f *fakePlatform) CreateProvider(context.Context, string, string, string, string, string, snapshot.ProviderCapabilities) (*Provider, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateProvider(context.Context, string, string, string, string) (*Provider, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteProvider(context.Context, string) error        { return nil }
func (f *fakePlatform) ListRoutes(context.Context, string) ([]Route, error) { return nil, nil }
func (f *fakePlatform) CreateRoute(context.Context, string, string, string, string, []RouteFallback) (*Route, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateRoute(context.Context, string, string, string, string, []RouteFallback) (*Route, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteRoute(context.Context, string) error { return nil }
func (f *fakePlatform) ListUsage(context.Context, string, UsageFilter) ([]UsageEvent, error) {
	return nil, nil
}
func (f *fakePlatform) SummarizeUsage(context.Context, string, UsageFilter) ([]UsageSummaryBucket, error) {
	return nil, nil
}
func (f *fakePlatform) ListQuotas(context.Context, string) ([]Quota, error) { return nil, nil }
func (f *fakePlatform) CreateQuota(context.Context, string, string, string, string, int64, string) (*Quota, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateQuota(context.Context, string, int64) (*Quota, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteQuota(context.Context, string) error { return nil }
func (f *fakePlatform) ListPolicies(context.Context, string) ([]RequestPolicy, error) {
	return nil, nil
}
func (f *fakePlatform) CreatePolicy(context.Context, string, string, string, string, []byte, bool, int) (*RequestPolicy, error) {
	return nil, nil
}
func (f *fakePlatform) UpdatePolicy(context.Context, string, *string, *string, *string, []byte, *bool, *int) (*RequestPolicy, error) {
	return nil, nil
}
func (f *fakePlatform) ReorderPolicies(context.Context, string, []gatewayconfig.PolicyPriorityUpdate) error {
	return nil
}
func (f *fakePlatform) DeletePolicy(context.Context, string) error { return nil }
func (f *fakePlatform) ListWasmHooks(context.Context, string) ([]WasmHook, error) {
	return nil, nil
}
func (f *fakePlatform) CreateWasmHook(context.Context, string, string, string, string, string, bool, int, []byte) (*WasmHook, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateWasmHook(context.Context, string, *string, *string, *string, *string, *bool, *int, []byte) (*WasmHook, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteWasmHook(context.Context, string) error { return nil }
func (f *fakePlatform) ListCredentials(context.Context, string) ([]Credential, error) {
	return nil, nil
}
func (f *fakePlatform) CreateCredential(context.Context, string, string, string, string, string, string) (*Credential, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) UpdateCredential(context.Context, string, string, string) (*Credential, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) RotateCredential(context.Context, string, string, string) (*Credential, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteCredential(context.Context, string) error { return nil }
func (f *fakePlatform) ListCredentialAssignments(context.Context, string) ([]CredentialAssignment, error) {
	return nil, nil
}
func (f *fakePlatform) AssignCredential(context.Context, string, string, string, string) (*CredentialAssignment, error) {
	return nil, errors.New("unused")
}
func (f *fakePlatform) DeleteCredentialAssignment(context.Context, string) error { return nil }

func testCfg() *kernel.Config {
	cfg := &kernel.Config{}
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Auth.TokenTTL = time.Hour
	cfg.Auth.InternalToken = "internal-secret"
	cfg.Mail.PublicAppURL = "http://localhost:3000"
	cfg.Mail.From = "AFI <noreply@afi.local>"
	cfg.Mail.DefaultProvider = "log"
	cfg.Mail.SMTP.Enabled = false
	cfg.Mail.Resend.Enabled = false
	return cfg
}

// testServer wires a fake that implements both thin platformAPI and platform.ConfigAPI.
func testServer(api interface {
	platformAPI
	platform.ConfigAPI
}, members membershipChecker) *Server {
	return &Server{
		cfg:       testCfg(),
		api:       api,
		config:    api,
		members:   members,
		publisher: &fakePublisher{},
		log:       slog.Default(),
	}
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
		api:       api, config: api,
		log: slog.Default(),
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

func TestCreateProviderRequiresAdmin(t *testing.T) {
	t.Parallel()
	api := &fakePlatform{fakeMembers: fakeMembers{
		allowed: map[string]bool{"user_1|org_a": true},
		admins:  map[string]bool{},
	}}
	s := &Server{
		cfg: testCfg(), api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_1", "a@b.c", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/providers",
		bytes.NewBufferString(`{"name":"OpenAI","type":"openai","base_url":"https://api.openai.com/v1","api_key_env":"OPENAI_API_KEY"}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateRouteRequiresAdmin(t *testing.T) {
	t.Parallel()
	api := &fakePlatform{fakeMembers: fakeMembers{
		allowed: map[string]bool{"user_1|org_a": true},
		admins:  map[string]bool{},
	}}
	s := &Server{
		cfg: testCfg(), api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_1", "a@b.c", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/organizations/org_a/routes",
		bytes.NewBufferString(`{"model":"gpt-4o","provider_id":"prov_ok","target_model":"gpt-4o"}`))
	req.SetPathValue("orgID", "org_a")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateProviderRequiresAdmin(t *testing.T) {
	t.Parallel()
	api := &fakePlatform{fakeMembers: fakeMembers{
		allowed: map[string]bool{"user_1|org_a": true},
		admins:  map[string]bool{},
	}}
	s := &Server{
		cfg: testCfg(), api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_1", "a@b.c", "member")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/providers/prov_ok",
		bytes.NewBufferString(`{"name":"Renamed"}`))
	req.SetPathValue("providerID", "prov_ok")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}

func TestUpdateOrgMemberRoleRequiresOwner(t *testing.T) {
	t.Parallel()
	api := &fakePlatform{fakeMembers: fakeMembers{
		allowed: map[string]bool{"user_admin|org_a": true},
		admins:  map[string]bool{"user_admin|org_a": true},
		owners:  map[string]bool{},
	}}
	s := &Server{
		cfg: testCfg(), api: api, config: api, members: api, publisher: &fakePublisher{}, log: slog.Default(),
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, time.Hour, "user_admin", "admin@afi.local", "admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/organizations/org_a/members/user_2",
		bytes.NewBufferString(`{"role":"admin"}`))
	req.SetPathValue("orgID", "org_a")
	req.SetPathValue("userID", "user_2")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}
