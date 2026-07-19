package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type Server struct {
	cfg       *kernel.Config
	api       platformAPI
	members   membershipChecker
	publisher snapshotPublisher
	seeder    *Seeder
	snapStore *snapshot.Store
	log       *slog.Logger
}

func NewServer(cfg *kernel.Config, store *Store, seeder *Seeder, snapStore *snapshot.Store, log *slog.Logger) *Server {
	return &Server{
		cfg:       cfg,
		api:       store,
		members:   store,
		publisher: seeder,
		seeder:    seeder,
		snapStore: snapStore,
		log:       log,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)

	mux.HandleFunc("POST /internal/v1/seed", s.requireInternal(s.handleSeed))
	mux.HandleFunc("POST /internal/v1/snapshots/publish", s.requireInternal(s.handlePublish))

	mux.HandleFunc("POST /api/v1/platform/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/v1/platform/auth/me", s.requireAuth(s.handleMe))

	mux.HandleFunc("GET /api/v1/platform/organizations", s.requireAuth(s.handleListOrgs))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/teams", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListTeams)))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/projects", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListProjects)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/projects", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleCreateProject)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/providers", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListProviders)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/providers", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleCreateProvider)))
	mux.HandleFunc("PATCH /api/v1/platform/providers/{providerID}", s.requireAuth(s.requireOrgMemberViaProvider(s.handleUpdateProvider)))
	mux.HandleFunc("DELETE /api/v1/platform/providers/{providerID}", s.requireAuth(s.requireOrgMemberViaProvider(s.handleDeleteProvider)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/routes", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListRoutes)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/routes", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleCreateRoute)))
	mux.HandleFunc("PATCH /api/v1/platform/routes/{routeID}", s.requireAuth(s.requireOrgMemberViaRoute(s.handleUpdateRoute)))
	mux.HandleFunc("DELETE /api/v1/platform/routes/{routeID}", s.requireAuth(s.requireOrgMemberViaRoute(s.handleDeleteRoute)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/usage", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListUsage)))

	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/quotas", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleListQuotas)))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/quotas", s.requireAuth(s.requireOrgMemberFromPath("orgID", s.handleCreateQuota)))
	mux.HandleFunc("PATCH /api/v1/platform/quotas/{quotaID}", s.requireAuth(s.requireOrgMemberViaQuota(s.handleUpdateQuota)))
	mux.HandleFunc("DELETE /api/v1/platform/quotas/{quotaID}", s.requireAuth(s.requireOrgMemberViaQuota(s.handleDeleteQuota)))

	mux.HandleFunc("GET /api/v1/platform/teams/{teamID}", s.requireAuth(s.requireOrgMemberViaTeam(s.handleGetTeam)))
	mux.HandleFunc("GET /api/v1/platform/teams/{teamID}/members", s.requireAuth(s.requireOrgMemberViaTeam(s.handleListTeamMembers)))

	mux.HandleFunc("GET /api/v1/platform/projects/{projectID}/keys", s.requireAuth(s.requireOrgMemberViaProject(s.handleListKeys)))
	mux.HandleFunc("POST /api/v1/platform/projects/{projectID}/keys", s.requireAuth(s.requireOrgMemberViaProject(s.handleCreateKey)))

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
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
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

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, err := s.api.GetUserByEmail(r.Context(), body.Email)
	if err != nil || !CheckPassword(user.PasswordHash, body.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	tok, err := IssueToken(s.cfg.Auth.JWTSecret, s.cfg.Auth.TokenTTL, user.ID, user.Email, user.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	user, err := s.api.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": user.ID, "name": user.Name, "email": user.Email, "role": user.Role,
	})
}

func (s *Server) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	orgs, err := s.api.ListOrganizationsForUser(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orgs == nil {
		orgs = []Organization{}
	}
	writeJSON(w, http.StatusOK, orgs)
}

func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := s.api.ListTeams(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if teams == nil {
		teams = []Team{}
	}
	writeJSON(w, http.StatusOK, teams)
}

func (s *Server) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	team, err := s.api.GetTeam(r.Context(), r.PathValue("teamID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, team)
}

func (s *Server) handleListTeamMembers(w http.ResponseWriter, r *http.Request) {
	members, err := s.api.ListTeamMembers(r.Context(), r.PathValue("teamID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if members == nil {
		members = []TeamMember{}
	}
	writeJSON(w, http.StatusOK, members)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.api.ListProjects(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		TeamID string `json:"team_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	p, err := s.api.CreateProject(r.Context(), r.PathValue("orgID"), body.TeamID, body.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "created but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.api.ListAPIKeys(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = []APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Key  string `json:"key"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" {
		body.Name = "API Key"
	}
	if body.Key == "" {
		body.Key = "sk-" + randomHex(24)
	}

	orgID, err := s.api.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}

	k, err := s.api.CreateAPIKey(r.Context(), orgID, r.PathValue("projectID"), body.Name, body.Key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "created but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	list, err := s.api.ListProviders(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Provider{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		BaseURL   string `json:"base_url"`
		APIKeyEnv string `json:"api_key_env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.BaseURL == "" {
		writeErr(w, http.StatusBadRequest, "name and base_url required")
		return
	}
	if body.Type == "" {
		body.Type = "openai"
	}
	if body.APIKeyEnv == "" {
		if body.Type == "anthropic" {
			body.APIKeyEnv = "ANTHROPIC_API_KEY"
		} else {
			body.APIKeyEnv = "OPENAI_API_KEY"
		}
	}
	p, err := s.api.CreateProvider(r.Context(), r.PathValue("orgID"), body.Name, body.Type, body.BaseURL, body.APIKeyEnv)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "created but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string `json:"name"`
		BaseURL   string `json:"base_url"`
		APIKeyEnv string `json:"api_key_env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.BaseURL == "" {
		writeErr(w, http.StatusBadRequest, "name and base_url required")
		return
	}
	if body.APIKeyEnv == "" {
		body.APIKeyEnv = "OPENAI_API_KEY"
	}
	p, err := s.api.UpdateProvider(r.Context(), r.PathValue("providerID"), body.Name, body.BaseURL, body.APIKeyEnv)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "updated but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := s.api.DeleteProvider(r.Context(), r.PathValue("providerID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "deleted but snapshot publish failed: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListRoutes(w http.ResponseWriter, r *http.Request) {
	list, err := s.api.ListRoutes(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Route{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model       string          `json:"model"`
		ProviderID  string          `json:"provider_id"`
		TargetModel string          `json:"target_model"`
		Fallbacks   []RouteFallback `json:"fallbacks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Model == "" || body.ProviderID == "" {
		writeErr(w, http.StatusBadRequest, "model and provider_id required")
		return
	}
	if body.TargetModel == "" {
		body.TargetModel = body.Model
	}
	route, err := s.api.CreateRoute(r.Context(), r.PathValue("orgID"), body.Model, body.ProviderID, body.TargetModel, body.Fallbacks)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "created but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, route)
}

func (s *Server) handleUpdateRoute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model       string          `json:"model"`
		ProviderID  string          `json:"provider_id"`
		TargetModel string          `json:"target_model"`
		Fallbacks   []RouteFallback `json:"fallbacks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Model == "" || body.ProviderID == "" {
		writeErr(w, http.StatusBadRequest, "model and provider_id required")
		return
	}
	if body.TargetModel == "" {
		body.TargetModel = body.Model
	}
	route, err := s.api.UpdateRoute(r.Context(), r.PathValue("routeID"), body.Model, body.ProviderID, body.TargetModel, body.Fallbacks)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "updated but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, route)
}

func (s *Server) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	if err := s.api.DeleteRoute(r.Context(), r.PathValue("routeID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "deleted but snapshot publish failed: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListUsage(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.api.ListUsage(r.Context(), r.PathValue("orgID"), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []UsageEvent{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleListQuotas(w http.ResponseWriter, r *http.Request) {
	list, err := s.api.ListQuotas(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Quota{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateQuota(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ScopeType  string `json:"scope_type"`
		ScopeID    string `json:"scope_id"`
		Metric     string `json:"metric"`
		LimitValue int64  `json:"limit_value"`
		Window     string `json:"window"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ScopeType == "" || body.ScopeID == "" || body.Metric == "" {
		writeErr(w, http.StatusBadRequest, "scope_type, scope_id, metric required")
		return
	}
	if body.LimitValue < 0 {
		writeErr(w, http.StatusBadRequest, "limit_value must be >= 0")
		return
	}
	q, err := s.api.CreateQuota(r.Context(), r.PathValue("orgID"), body.ScopeType, body.ScopeID, body.Metric, body.LimitValue, body.Window)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "created but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, q)
}

func (s *Server) handleUpdateQuota(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LimitValue int64 `json:"limit_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LimitValue < 0 {
		writeErr(w, http.StatusBadRequest, "limit_value required (>= 0)")
		return
	}
	q, err := s.api.UpdateQuota(r.Context(), r.PathValue("quotaID"), body.LimitValue)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "updated but snapshot publish failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) handleDeleteQuota(w http.ResponseWriter, r *http.Request) {
	if err := s.api.DeleteQuota(r.Context(), r.PathValue("quotaID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.publisher.PublishSnapshot(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, "deleted but snapshot publish failed: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type ctxClaimsKey int

const claimsKey ctxClaimsKey = 1

func claimsFrom(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		claims, err := ParseToken(s.cfg.Auth.JWTSecret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
	}
}

func (s *Server) requireInternal(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := CheckInternalToken(s.cfg.Auth.InternalToken, r.Header.Get("X-AFI-Internal-Token")); err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberFromPath(pathKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		orgID := r.PathValue(pathKey)
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberViaTeam(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetTeamOrgID(r.Context(), r.PathValue("teamID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberViaProject(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberViaProvider(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProviderOrgID(r.Context(), r.PathValue("providerID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberViaRoute(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetRouteOrgID(r.Context(), r.PathValue("routeID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireOrgMemberViaQuota(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetQuotaOrgID(r.Context(), r.PathValue("quotaID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
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
