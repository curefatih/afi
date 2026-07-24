package controlplane

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envs, err := s.app.ListEnvironments(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if envs == nil {
		envs = []Environment{}
	}
	writeJSON(w, http.StatusOK, envs)
}

func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	if body.Slug == "" {
		writeErr(w, http.StatusBadRequest, "slug required")
		return
	}
	orgID, err := s.members.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}
	e, err := s.app.CreateEnvironment(r.Context(), orgID, r.PathValue("projectID"), body.Name, body.Slug)
	if err != nil {
		if errors.Is(err, kernel.ErrInvalidRequest) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	e, err := s.app.GetEnvironment(r.Context(), r.PathValue("environmentID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteEnvironment(r.Context(), r.PathValue("environmentID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
