package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
)

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
	member, err := s.app.UpdateTeamMemberRole(
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
