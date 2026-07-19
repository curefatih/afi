package controlplane

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
)

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
