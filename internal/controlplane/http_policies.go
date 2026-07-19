package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

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
