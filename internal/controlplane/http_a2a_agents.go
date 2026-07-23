package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListA2AAgents(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListA2AAgents(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []A2AAgent{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateA2AAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Alias      string          `json:"alias"`
		Name       string          `json:"name"`
		UpstreamURL string         `json:"upstream_url"`
		CardURL    string          `json:"card_url"`
		CardCache  json.RawMessage `json:"card_cache"`
		APIKeyEnv  string          `json:"api_key_env"`
		AuthScheme string          `json:"auth_scheme"`
		Enabled    *bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Alias == "" || body.Name == "" || body.UpstreamURL == "" {
		writeErr(w, http.StatusBadRequest, "alias, name, and upstream_url required")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	a, err := s.app.CreateA2AAgent(r.Context(), r.PathValue("orgID"), body.Alias, body.Name, body.UpstreamURL, body.CardURL, body.APIKeyEnv, body.AuthScheme, body.CardCache, enabled)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) handleUpdateA2AAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Alias       *string         `json:"alias"`
		Name        *string         `json:"name"`
		UpstreamURL *string         `json:"upstream_url"`
		CardURL     *string         `json:"card_url"`
		CardCache   json.RawMessage `json:"card_cache"`
		APIKeyEnv   *string         `json:"api_key_env"`
		AuthScheme  *string         `json:"auth_scheme"`
		Enabled     *bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Alias == nil && body.Name == nil && body.UpstreamURL == nil && body.CardURL == nil &&
		body.CardCache == nil && body.APIKeyEnv == nil && body.AuthScheme == nil && body.Enabled == nil {
		writeErr(w, http.StatusBadRequest, "at least one field required")
		return
	}
	a, err := s.app.UpdateA2AAgent(r.Context(), r.PathValue("agentID"), body.Alias, body.Name, body.UpstreamURL, body.CardURL, body.APIKeyEnv, body.AuthScheme, body.CardCache, body.Enabled)
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

func (s *Server) handleDeleteA2AAgent(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteA2AAgent(r.Context(), r.PathValue("agentID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
