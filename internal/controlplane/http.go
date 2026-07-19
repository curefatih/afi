package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/mail"
	"github.com/curefatih/afi/internal/snapshot"
)

type Server struct {
	cfg         *kernel.Config
	api         platformAPI
	app         *platform.Service
	members     membershipChecker
	publisher   snapshotPublisher
	seeder      *Seeder
	snapStore   snapshot.Store
	log         *slog.Logger
	eventOutbox platform.EventEnqueuer
}

func NewServer(cfg *kernel.Config, store *Store, seeder *Seeder, snapStore snapshot.Store, log *slog.Logger, eventOutbox platform.EventEnqueuer) *Server {
	app := platform.New(store, seeder)
	app.Events = newPlatformEventBus(log, eventOutbox)
	return &Server{
		cfg:         cfg,
		api:         store,
		app:         app,
		members:     store,
		publisher:   seeder,
		seeder:      seeder,
		snapStore:   snapStore,
		log:         log,
		eventOutbox: eventOutbox,
	}
}

// configAPIAdapter lets platformAPI (used by tests) satisfy platform.ConfigAPI.
type configAPIAdapter struct{ platformAPI }

func (s *Server) ensureApp() {
	if s.app != nil || s.api == nil {
		return
	}
	s.app = platform.New(configAPIAdapter{s.api}, s.publisher)
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
	mux.HandleFunc("PATCH /api/v1/platform/policies/{policyID}", s.requireAuth(s.requireOrgAdminViaPolicy(s.handleUpdatePolicy)))
	mux.HandleFunc("DELETE /api/v1/platform/policies/{policyID}", s.requireAuth(s.requireOrgAdminViaPolicy(s.handleDeletePolicy)))

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
	mux.HandleFunc("PATCH /api/v1/platform/teams/{teamID}/members/{userID}", s.requireAuth(s.requireTeamManager(s.handleUpdateTeamMemberRole)))
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
	orgs, err := s.app.ListOrganizationsForUser(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orgs == nil {
		orgs = []Organization{}
	}
	writeJSON(w, http.StatusOK, orgs)
}

func (s *Server) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	org, err := s.app.CreateOrganization(r.Context(), strings.TrimSpace(body.Name), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (s *Server) handleListOrgMembers(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListOrgMembers(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []OrgMember{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleInviteOrgMember(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Email) == "" {
		writeErr(w, http.StatusBadRequest, "email required")
		return
	}
	orgID := r.PathValue("orgID")
	outcome, rawToken, err := s.app.InviteOrgMember(r.Context(), orgID, strings.TrimSpace(body.Email), claims.UserID)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.sendInviteMail(r.Context(), orgID, outcome, rawToken); err != nil {
		s.log.Error("invite mail", "err", err, "org", orgID)
		writeErr(w, http.StatusBadGateway, "member updated but email failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, outcome)
}

func (s *Server) handleListOrgInvites(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListOrgInvites(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []OrgInvite{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleRevokeOrgInvite(w http.ResponseWriter, r *http.Request) {
	err := s.app.RevokeOrgInvite(r.Context(), r.PathValue("orgID"), r.PathValue("inviteID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "invalid invite")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleResendOrgInvite(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgID")
	inv, rawToken, err := s.app.ResendOrgInvite(r.Context(), orgID, r.PathValue("inviteID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "invalid invite")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	outcome := &InviteOutcome{Status: "invited", Invite: inv}
	if err := s.sendInviteMail(r.Context(), orgID, outcome, rawToken); err != nil {
		writeErr(w, http.StatusBadGateway, "email failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (s *Server) handlePreviewInvite(w http.ResponseWriter, r *http.Request) {
	preview, err := s.app.PreviewOrgInvite(r.Context(), r.PathValue("token"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "invite not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "invite expired")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)

	preview, err := s.app.PreviewOrgInvite(r.Context(), r.PathValue("token"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "invite not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "invite expired")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	passwordHash := ""
	if !preview.UserExists {
		if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Password) == "" {
			writeErr(w, http.StatusBadRequest, "name and password required")
			return
		}
		if len(body.Password) < 8 {
			writeErr(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		h, err := HashPassword(body.Password)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		passwordHash = h
	}

	member, user, err := s.app.AcceptOrgInvite(r.Context(), r.PathValue("token"), strings.TrimSpace(body.Name), passwordHash)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "invite not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	tok, err := IssueToken(s.cfg.Auth.JWTSecret, s.cfg.Auth.TokenTTL, user.ID, user.Email, user.Role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"member": member,
		"user":   map[string]any{"id": user.ID, "email": user.Email, "name": user.Name, "role": user.Role},
		"token":  tok,
	})
}

func (s *Server) handleGetOrgMail(w http.ResponseWriter, r *http.Request) {
	org, err := s.api.GetOrganization(r.Context(), r.PathValue("orgID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"selected":          org.MailProvider,
		"default_provider":  s.cfg.Mail.DefaultProvider,
		"enabled_providers": enabledMailProviders(s.cfg),
		"from":              s.cfg.Mail.From,
		"public_app_url":    s.cfg.Mail.PublicAppURL,
	})
}

func (s *Server) handleUpdateOrgMail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider *string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Provider == nil {
		writeErr(w, http.StatusBadRequest, "provider required")
		return
	}
	provider := mail.ProviderName(*body.Provider)
	if provider != "" {
		enabled := false
		for _, p := range enabledMailProviders(s.cfg) {
			if p == provider {
				enabled = true
				break
			}
		}
		if !enabled {
			writeErr(w, http.StatusBadRequest, "provider is not enabled on this deployment")
			return
		}
	}
	org, err := s.api.SetOrgMailProvider(r.Context(), r.PathValue("orgID"), provider)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "invalid provider")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"selected":          org.MailProvider,
		"default_provider":  s.cfg.Mail.DefaultProvider,
		"enabled_providers": enabledMailProviders(s.cfg),
	})
}

func (s *Server) handleTestOrgMail(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	user, err := s.api.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	org, err := s.api.GetOrganization(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	sender, _, err := resolveMailSender(s.cfg, org.MailProvider, s.log)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	msg := mail.TestMessage()
	msg.To = user.Email
	if err := sender.Send(r.Context(), msg); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "to": user.Email})
}

func (s *Server) sendInviteMail(ctx context.Context, orgID string, outcome *InviteOutcome, rawToken string) error {
	if outcome == nil {
		return nil
	}
	org, err := s.api.GetOrganization(ctx, orgID)
	if err != nil {
		return err
	}
	sender, _, err := resolveMailSender(s.cfg, org.MailProvider, s.log)
	if err != nil {
		return err
	}
	base := strings.TrimRight(s.cfg.Mail.PublicAppURL, "/")
	var msg mail.Message
	switch outcome.Status {
	case "added":
		if outcome.Member == nil {
			return nil
		}
		msg = mail.InviteExistingUser(org.Name, base+"/auth/login")
		msg.To = outcome.Member.Email
	case "invited":
		if outcome.Invite == nil || rawToken == "" {
			return nil
		}
		msg = mail.InviteNewUser(org.Name, base+"/auth/invite/"+rawToken)
		msg.To = outcome.Invite.Email
	default:
		return nil
	}
	return sender.Send(ctx, msg)
}

func (s *Server) handleUpdateOrgMemberRole(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Role) == "" {
		writeErr(w, http.StatusBadRequest, "role required")
		return
	}
	member, err := s.app.UpdateOrgMemberRole(
		r.Context(),
		r.PathValue("orgID"),
		claims.UserID,
		r.PathValue("userID"),
		strings.TrimSpace(body.Role),
	)
	if errors.Is(err, kernel.ErrUnauthorized) {
		writeErr(w, http.StatusForbidden, "only the org owner can change roles")
		return
	}
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "member not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	teams, err := s.app.ListTeams(r.Context(), r.PathValue("orgID"), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if teams == nil {
		teams = []Team{}
	}
	writeJSON(w, http.StatusOK, teams)
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	team, err := s.app.CreateTeam(r.Context(), r.PathValue("orgID"), strings.TrimSpace(body.Name), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, team)
}

func (s *Server) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	team, err := s.app.GetTeam(r.Context(), r.PathValue("teamID"))
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
	members, err := s.app.ListTeamMembers(r.Context(), r.PathValue("teamID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if members == nil {
		members = []TeamMember{}
	}
	writeJSON(w, http.StatusOK, members)
}

func (s *Server) handleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.UserID) == "" {
		writeErr(w, http.StatusBadRequest, "user_id required")
		return
	}
	member, err := s.app.AddTeamMember(r.Context(), r.PathValue("teamID"), strings.TrimSpace(body.UserID))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

func (s *Server) handleUpdateTeamMemberRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Role) == "" {
		writeErr(w, http.StatusBadRequest, "role required")
		return
	}
	member, err := s.api.UpdateTeamMemberRole(
		r.Context(),
		r.PathValue("teamID"),
		r.PathValue("userID"),
		strings.TrimSpace(body.Role),
	)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "member not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) handleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	err := s.app.RemoveTeamMember(r.Context(), r.PathValue("teamID"), r.PathValue("userID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, "cannot remove the sole team owner")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	projects, err := s.app.ListProjects(r.Context(), r.PathValue("orgID"), claims.UserID)
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
	p, err := s.app.CreateProject(r.Context(), r.PathValue("orgID"), body.TeamID, body.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.app.ListAPIKeys(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = []APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleListOrgKeys(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgID")
	claims := claimsFrom(r.Context())
	admin, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	keys, err := s.app.ListVisibleOrgAPIKeys(r.Context(), orgID, claims.UserID, admin)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = []APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleCreateOrgKey(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgID")
	claims := claimsFrom(r.Context())
	var body struct {
		Name      string `json:"name"`
		Key       string `json:"key"`
		Kind      string `json:"kind"`
		ProjectID string `json:"project_id"`
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
	if body.Kind == "" {
		body.Kind = snapshot.KeyKindPersonal
	}

	ownerUserID := ""
	switch body.Kind {
	case snapshot.KeyKindPersonal:
		if body.ProjectID != "" {
			writeErr(w, http.StatusBadRequest, "personal keys cannot have a project")
			return
		}
		ownerUserID = claims.UserID
	case snapshot.KeyKindServiceAccount:
		admin, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !admin {
			writeErr(w, http.StatusForbidden, "only org admins can create service account keys")
			return
		}
	default:
		writeErr(w, http.StatusBadRequest, "kind must be personal or service_account")
		return
	}

	k, err := s.app.CreateAPIKey(r.Context(), orgID, body.Kind, ownerUserID, body.ProjectID, body.Name, body.Key)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	keyID := r.PathValue("keyID")
	k, err := s.api.GetAPIKey(r.Context(), keyID)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, k.OrganizationID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	admin, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, k.OrganizationID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !admin && !(k.Kind == snapshot.KeyKindPersonal && k.OwnerUserID == claims.UserID) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := s.app.DeleteAPIKey(r.Context(), keyID); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

	k, err := s.app.CreateAPIKey(r.Context(), orgID, snapshot.KeyKindServiceAccount, "", r.PathValue("projectID"), body.Name, body.Key)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListProviders(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Provider{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	f, err := parseUsageFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	from := time.Time{}
	to := time.Time{}
	if f.From != nil {
		from = *f.From
	}
	if f.To != nil {
		to = *f.To
	}
	list, err := s.app.ListProviderHealth(r.Context(), r.PathValue("orgID"), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []ProviderHealth{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string                        `json:"name"`
		Type         string                        `json:"type"`
		BaseURL      string                        `json:"base_url"`
		APIKeyEnv    string                        `json:"api_key_env"`
		Capabilities snapshot.ProviderCapabilities `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.BaseURL == "" {
		writeErr(w, http.StatusBadRequest, "name and base_url required")
		return
	}
	if body.Type == "" {
		body.Type = "openai"
	}
	if body.APIKeyEnv == "" {
		body.APIKeyEnv = snapshot.DefaultAPIKeyEnv(body.Type)
	}
	p, err := s.app.CreateProvider(r.Context(), r.PathValue("orgID"), body.Name, body.Type, body.BaseURL, body.APIKeyEnv, body.Capabilities)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
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
	p, err := s.app.UpdateProvider(r.Context(), r.PathValue("providerID"), body.Name, body.BaseURL, body.APIKeyEnv)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteProvider(r.Context(), r.PathValue("providerID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListRoutes(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListRoutes(r.Context(), r.PathValue("orgID"))
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
	route, err := s.app.CreateRoute(r.Context(), r.PathValue("orgID"), body.Model, body.ProviderID, body.TargetModel, body.Fallbacks)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
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
	route, err := s.app.UpdateRoute(r.Context(), r.PathValue("routeID"), body.Model, body.ProviderID, body.TargetModel, body.Fallbacks)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, route)
}

func (s *Server) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteRoute(r.Context(), r.PathValue("routeID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseUsageFilter(r *http.Request) (UsageFilter, error) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := UsageFilter{
		Limit:     limit,
		ProjectID: q.Get("project_id"),
		APIKeyID:  q.Get("api_key_id"),
		Model:     q.Get("model"),
		Modality:  q.Get("modality"),
		GroupBy:   q.Get("group_by"),
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return f, fmt.Errorf("%w: invalid from", kernel.ErrInvalidRequest)
			}
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
			if err != nil {
				return f, fmt.Errorf("%w: invalid to", kernel.ErrInvalidRequest)
			}
		}
		f.To = &t
	}
	return f, nil
}

func (s *Server) handleListUsage(w http.ResponseWriter, r *http.Request) {
	f, err := parseUsageFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := s.app.ListUsage(r.Context(), r.PathValue("orgID"), f)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []UsageEvent{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleUsageSummary(w http.ResponseWriter, r *http.Request) {
	f, err := parseUsageFilter(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if f.GroupBy == "" {
		f.GroupBy = "day"
	}
	list, err := s.app.SummarizeUsage(r.Context(), r.PathValue("orgID"), f)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []UsageSummaryBucket{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleListQuotas(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListQuotas(r.Context(), r.PathValue("orgID"))
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
	q, err := s.app.CreateQuota(r.Context(), r.PathValue("orgID"), body.ScopeType, body.ScopeID, body.Metric, body.LimitValue, body.Window)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
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
	q, err := s.app.UpdateQuota(r.Context(), r.PathValue("quotaID"), body.LimitValue)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) handleDeleteQuota(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteQuota(r.Context(), r.PathValue("quotaID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListPolicies(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []RequestPolicy{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       string `json:"name"`
		Expression string `json:"expression"`
		Enabled    *bool  `json:"enabled"`
		Priority   *int   `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Expression == "" {
		writeErr(w, http.StatusBadRequest, "name and expression required")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	priority := 100
	if body.Priority != nil {
		priority = *body.Priority
	}
	p, err := s.app.CreatePolicy(r.Context(), r.PathValue("orgID"), body.Name, body.Expression, enabled, priority)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       *string `json:"name"`
		Expression *string `json:"expression"`
		Enabled    *bool   `json:"enabled"`
		Priority   *int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == nil && body.Expression == nil && body.Enabled == nil && body.Priority == nil {
		writeErr(w, http.StatusBadRequest, "at least one field required")
		return
	}
	p, err := s.app.UpdatePolicy(r.Context(), r.PathValue("policyID"), body.Name, body.Expression, body.Enabled, body.Priority)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeletePolicy(r.Context(), r.PathValue("policyID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListCredentials(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Credential{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		ProviderType string `json:"provider_type"`
		StorageKind  string `json:"storage_kind"`
		SecretRef    string `json:"secret_ref"`
		SecretValue  string `json:"secret_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.ProviderType == "" || body.StorageKind == "" {
		writeErr(w, http.StatusBadRequest, "name, provider_type, and storage_kind required")
		return
	}
	c, err := s.app.CreateCredential(r.Context(), r.PathValue("orgID"), body.Name, body.ProviderType, body.StorageKind, body.SecretRef, body.SecretValue)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" && body.Status == "" {
		writeErr(w, http.StatusBadRequest, "name or status required")
		return
	}
	c, err := s.app.UpdateCredential(r.Context(), r.PathValue("credentialID"), body.Name, body.Status)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleRotateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SecretRef   string `json:"secret_ref"`
		SecretValue string `json:"secret_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	c, err := s.app.RotateCredential(r.Context(), r.PathValue("credentialID"), body.SecretRef, body.SecretValue)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	err := s.app.DeleteCredential(r.Context(), r.PathValue("credentialID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrConflict) {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListCredentialAssignments(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListCredentialAssignments(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []CredentialAssignment{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAssignCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CredentialID string `json:"credential_id"`
		ScopeType    string `json:"scope_type"`
		ScopeID      string `json:"scope_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CredentialID == "" || body.ScopeType == "" || body.ScopeID == "" {
		writeErr(w, http.StatusBadRequest, "credential_id, scope_type, and scope_id required")
		return
	}
	orgID := r.PathValue("orgID")
	credOrg, err := s.members.GetCredentialOrgID(r.Context(), body.CredentialID)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if credOrg != orgID {
		writeErr(w, http.StatusBadRequest, "credential does not belong to organization")
		return
	}
	claims := claimsFrom(r.Context())
	a, err := s.app.AssignCredential(r.Context(), body.CredentialID, body.ScopeType, body.ScopeID, claims.UserID)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleDeleteCredentialAssignment(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteCredentialAssignment(r.Context(), r.PathValue("assignmentID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
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
		s.ensureApp()
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

func (s *Server) requireOrgAdminFromPath(pathKey string, next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberFromPath(pathKey, func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		orgID := r.PathValue(pathKey)
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgOwnerFromPath(pathKey string, next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberFromPath(pathKey, func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		orgID := r.PathValue(pathKey)
		ok, err := s.members.IsOrgOwner(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgAdminViaProject(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaProject(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgAdminViaQuota(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaQuota(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetQuotaOrgID(r.Context(), r.PathValue("quotaID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}

func (s *Server) requireOrgAdminViaPolicy(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetPolicyOrgID(r.Context(), r.PathValue("policyID"))
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
		ok, err = s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
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

func (s *Server) requireOrgAdminViaCredential(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetCredentialOrgID(r.Context(), r.PathValue("credentialID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
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

func (s *Server) requireOrgAdminViaCredentialAssignment(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetCredentialAssignmentOrgID(r.Context(), r.PathValue("assignmentID"))
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
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

func (s *Server) requireTeamAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamID := r.PathValue("teamID")
		_, err := s.members.GetTeamOrgID(r.Context(), teamID)
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.CanAccessTeam(r.Context(), teamID, claims.UserID)
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

func (s *Server) requireTeamManager(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamID := r.PathValue("teamID")
		_, err := s.members.GetTeamOrgID(r.Context(), teamID)
		if errors.Is(err, kernel.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.CanManageTeam(r.Context(), teamID, claims.UserID)
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

func (s *Server) requireOrgAdminViaProvider(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaProvider(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetProviderOrgID(r.Context(), r.PathValue("providerID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
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

func (s *Server) requireOrgAdminViaRoute(next http.HandlerFunc) http.HandlerFunc {
	return s.requireOrgMemberViaRoute(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := s.members.GetRouteOrgID(r.Context(), r.PathValue("routeID"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		claims := claimsFrom(r.Context())
		ok, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
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
