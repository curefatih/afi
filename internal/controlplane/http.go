package controlplane

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type Server struct {
	cfg         *kernel.Config
	api         platformAPI
	config      platform.ConfigAPI // app persistence; used by ensureApp when app is nil
	app         *platform.Service
	auth        *platform.AuthService
	members     membershipChecker
	publisher   snapshotPublisher
	seeder      *Seeder
	snapStore   snapshot.Store
	log         *slog.Logger
	eventOutbox platform.EventEnqueuer
}

func NewServer(cfg *kernel.Config, store *Store, seeder *Seeder, snapStore snapshot.Store, log *slog.Logger, eventOutbox platform.EventEnqueuer, ssoStates identity.SSOStateStore) *Server {
	app := platform.New(store, seeder)
	app.Events = newPlatformEventBus(log, eventOutbox)
	return &Server{
		cfg:         cfg,
		api:         store,
		config:      store,
		app:         app,
		auth:        newAuthService(cfg, store, ssoStates),
		members:     store,
		publisher:   seeder,
		seeder:      seeder,
		snapStore:   snapStore,
		log:         log,
		eventOutbox: eventOutbox,
	}
}

func (s *Server) ensureApp() {
	if s.app != nil || s.config == nil {
		return
	}
	s.app = platform.New(s.config, s.publisher)
	s.app.Events = newPlatformEventBus(s.log, s.eventOutbox)
}

func (s *Server) Handler() http.Handler {
	s.ensureApp()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)

	mux.HandleFunc("POST /internal/v1/seed", s.requireInternal(s.handleSeed))
	mux.HandleFunc("POST /internal/v1/snapshots/publish", s.requireInternal(s.handlePublish))

	mux.HandleFunc("POST /api/v1/platform/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/v1/platform/auth/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("GET /api/v1/platform/auth/sso/providers", s.handleListSSOProviders)
	mux.HandleFunc("GET /api/v1/platform/auth/sso/{provider}/start", s.handleSSOStart)
	mux.HandleFunc("GET /api/v1/platform/auth/sso/{provider}/callback", s.handleSSOCallback)
	mux.HandleFunc("GET /api/v1/platform/auth/invites/{token}", s.handlePreviewInvite)
	mux.HandleFunc("POST /api/v1/platform/auth/invites/{token}/accept", s.handleAcceptInvite)

	mux.HandleFunc("GET /api/v1/platform/organizations", s.requireAuth(s.handleListOrgs))
	mux.HandleFunc("POST /api/v1/platform/organizations", s.requireAuth(s.handleCreateOrg))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/members", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListOrgMembers)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/members", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleInviteOrgMember)))
	mux.HandleFunc("PATCH /api/v1/platform/organizations/{orgID}/members/{userID}", s.requireAuth(s.requireOrgOwnerFromPath("orgID", s.handleUpdateOrgMemberRole)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/invites", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleListOrgInvites)))
	mux.HandleFunc("DELETE /api/v1/platform/organizations/{orgID}/invites/{inviteID}", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleRevokeOrgInvite)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/invites/{inviteID}/resend", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleResendOrgInvite)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/mail", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleGetOrgMail)))
	mux.HandleFunc("PATCH /api/v1/platform/organizations/{orgID}/mail", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleUpdateOrgMail)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/mail/test", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleTestOrgMail)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/default-retry", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleGetOrgDefaultRetry)))
	mux.HandleFunc("PUT /api/v1/platform/organizations/{orgID}/default-retry", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleUpdateOrgDefaultRetry)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/teams", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListTeams)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/teams", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreateTeam)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/projects", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListProjects)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/projects", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleCreateProject)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/providers", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListProviders)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/providers/health", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleProviderHealth)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/providers", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreateProvider)))
	mux.HandleFunc("PATCH /api/v1/platform/providers/{providerID}", s.requireAuth(s.requireOrgAdminViaProvider(s.handleUpdateProvider)))
	mux.HandleFunc("DELETE /api/v1/platform/providers/{providerID}", s.requireAuth(s.requireOrgAdminViaProvider(s.handleDeleteProvider)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/routes", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListRoutes)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/routes", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreateRoute)))
	mux.HandleFunc("PATCH /api/v1/platform/routes/{routeID}", s.requireAuth(s.requireOrgAdminViaRoute(s.handleUpdateRoute)))
	mux.HandleFunc("DELETE /api/v1/platform/routes/{routeID}", s.requireAuth(s.requireOrgAdminViaRoute(s.handleDeleteRoute)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/usage", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListUsage)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/usage/summary", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleUsageSummary)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/quotas", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListQuotas)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/quotas", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreateQuota)))
	mux.HandleFunc("PATCH /api/v1/platform/quotas/{quotaID}", s.requireAuth(s.requireOrgAdminViaQuota(s.handleUpdateQuota)))
	mux.HandleFunc("DELETE /api/v1/platform/quotas/{quotaID}", s.requireAuth(s.requireOrgAdminViaQuota(s.handleDeleteQuota)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/policies", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListPolicies)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/policies", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreatePolicy)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/policies/reorder", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleReorderPolicies)))
	mux.HandleFunc("PATCH /api/v1/platform/policies/{policyID}", s.requireAuth(s.requireOrgAdminViaPolicy(s.handleUpdatePolicy)))
	mux.HandleFunc("DELETE /api/v1/platform/policies/{policyID}", s.requireAuth(s.requireOrgAdminViaPolicy(s.handleDeletePolicy)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/wasm-hooks", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListWasmHooks)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/wasm-hooks", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreateWasmHook)))
	mux.HandleFunc("PATCH /api/v1/platform/wasm-hooks/{hookID}", s.requireAuth(s.requireOrgAdminViaWasmHook(s.handleUpdateWasmHook)))
	mux.HandleFunc("DELETE /api/v1/platform/wasm-hooks/{hookID}", s.requireAuth(s.requireOrgAdminViaWasmHook(s.handleDeleteWasmHook)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/mcp-backends", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListMCPBackends)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/mcp-backends", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreateMCPBackend)))
	mux.HandleFunc("PATCH /api/v1/platform/mcp-backends/{backendID}", s.requireAuth(s.requireOrgAdminViaMCPBackend(s.handleUpdateMCPBackend)))
	mux.HandleFunc("DELETE /api/v1/platform/mcp-backends/{backendID}", s.requireAuth(s.requireOrgAdminViaMCPBackend(s.handleDeleteMCPBackend)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/credentials", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListCredentials)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/credentials", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleCreateCredential)))
	mux.HandleFunc("PATCH /api/v1/platform/credentials/{credentialID}", s.requireAuth(s.requireOrgAdminViaCredential(s.handleUpdateCredential)))
	mux.HandleFunc("POST /api/v1/platform/credentials/{credentialID}/rotate", s.requireAuth(s.requireOrgAdminViaCredential(s.handleRotateCredential)))
	mux.HandleFunc("DELETE /api/v1/platform/credentials/{credentialID}", s.requireAuth(s.requireOrgAdminViaCredential(s.handleDeleteCredential)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/credential-assignments", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListCredentialAssignments)))
	mux.HandleFunc("PUT /api/v1/platform/organizations/{orgID}/credential-assignments", s.requireAuth(s.requireOrgAdminFromPath("orgID", s.handleAssignCredential)))
	mux.HandleFunc("DELETE /api/v1/platform/credential-assignments/{assignmentID}", s.requireAuth(s.requireOrgAdminViaCredentialAssignment(s.handleDeleteCredentialAssignment)))

	mux.HandleFunc("GET /api/v1/platform/teams/{teamID}", s.requireAuth(s.requireTeamAccess(s.handleGetTeam)))
	mux.HandleFunc("GET /api/v1/platform/teams/{teamID}/members", s.requireAuth(s.requireTeamAccess(s.handleListTeamMembers)))
	mux.HandleFunc("POST /api/v1/platform/teams/{teamID}/members", s.requireAuth(s.requireTeamManager(s.handleAddTeamMember)))
	mux.HandleFunc("PATCH /api/v1/platform/teams/{teamID}/members/{userID}", s.requireAuth(s.requireTeamRoleChanger(s.handleUpdateTeamMemberRole)))
	mux.HandleFunc("DELETE /api/v1/platform/teams/{teamID}/members/{userID}", s.requireAuth(s.requireTeamManager(s.handleRemoveTeamMember)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/keys", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListOrgKeys)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/keys", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleCreateOrgKey)))
	mux.HandleFunc("DELETE /api/v1/platform/keys/{keyID}", s.requireAuth(s.handleDeleteKey))

	mux.HandleFunc("GET /api/v1/platform/projects/{projectID}/keys", s.requireAuth(s.requireOrgMemberViaProject(s.handleListKeys)))
	mux.HandleFunc("POST /api/v1/platform/projects/{projectID}/keys", s.requireAuth(s.requireOrgAdminViaProject(s.handleCreateKey)))

	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSeed(w http.ResponseWriter, r *http.Request) {
	if s.seeder == nil {
		writeErr(w, http.StatusInternalServerError, "seeder unavailable")
		return
	}
	if err := s.seeder.Seed(r.Context()); err != nil {
		s.log.Error("seed failed", "err", err)
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "seeded"})
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if err := s.app.PublishSnapshot(r.Context()); err != nil {
		s.log.Error("publish failed", "err", err)
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s.snapStore == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "published"})
		return
	}
	snap, err := s.snapStore.Latest(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"version": snap.Version})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-AFI-Internal-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
