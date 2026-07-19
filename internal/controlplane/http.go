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
	"strings"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type Server struct {
	cfg       *kernel.Config
	store     *Store
	seeder    *Seeder
	snapStore *snapshot.Store
	log       *slog.Logger
}

func NewServer(cfg *kernel.Config, store *Store, seeder *Seeder, snapStore *snapshot.Store, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: store, seeder: seeder, snapStore: snapStore, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)

	mux.HandleFunc("POST /internal/v1/seed", s.handleSeed)
	mux.HandleFunc("POST /internal/v1/snapshots/publish", s.handlePublish)

	mux.HandleFunc("POST /api/v1/platform/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/v1/platform/auth/me", s.requireAuth(s.handleMe))

	mux.HandleFunc("GET /api/v1/platform/organizations", s.requireAuth(s.handleListOrgs))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/teams", s.requireAuth(s.handleListTeams))
	mux.HandleFunc("GET /api/v1/platform/organizations/{orgID}/projects", s.requireAuth(s.handleListProjects))
	mux.HandleFunc("POST /api/v1/platform/organizations/{orgID}/projects", s.requireAuth(s.handleCreateProject))

	mux.HandleFunc("GET /api/v1/platform/teams/{teamID}", s.requireAuth(s.handleGetTeam))
	mux.HandleFunc("GET /api/v1/platform/teams/{teamID}/members", s.requireAuth(s.handleListTeamMembers))

	mux.HandleFunc("GET /api/v1/platform/projects/{projectID}/keys", s.requireAuth(s.handleListKeys))
	mux.HandleFunc("POST /api/v1/platform/projects/{projectID}/keys", s.requireAuth(s.handleCreateKey))

	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSeed(w http.ResponseWriter, r *http.Request) {
	if err := s.seeder.Seed(r.Context()); err != nil {
		s.log.Error("seed failed", "err", err)
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "seeded"})
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if err := s.seeder.PublishSnapshot(r.Context()); err != nil {
		s.log.Error("publish failed", "err", err)
		writeErr(w, http.StatusInternalServerError, err.Error())
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
	user, err := s.store.GetUserByEmail(r.Context(), body.Email)
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
	user, err := s.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

func (s *Server) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	orgs, err := s.store.ListOrganizationsForUser(r.Context(), claims.UserID)
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
	teams, err := s.store.ListTeams(r.Context(), r.PathValue("orgID"))
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
	team, err := s.store.GetTeam(r.Context(), r.PathValue("teamID"))
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
	members, err := s.store.ListTeamMembers(r.Context(), r.PathValue("teamID"))
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
	projects, err := s.store.ListProjects(r.Context(), r.PathValue("orgID"))
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
	p, err := s.store.CreateProject(r.Context(), r.PathValue("orgID"), body.TeamID, body.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.seeder.PublishSnapshot(r.Context())
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.ListAPIKeys(r.Context(), r.PathValue("projectID"))
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

	orgID, err := s.store.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}

	k, err := s.store.CreateAPIKey(r.Context(), orgID, r.PathValue("projectID"), body.Name, body.Key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.seeder.PublishSnapshot(r.Context())
	writeJSON(w, http.StatusCreated, k)
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
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next(w, r.WithContext(ctx))
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
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
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
