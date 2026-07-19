package controlplane

import (
	"encoding/json"
	"net/http"
)

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
