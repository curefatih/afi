package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListMCPBackends(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListMCPBackends(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []MCPBackend{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateMCPBackend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Alias           string          `json:"alias"`
		Name            string          `json:"name"`
		BaseURL         string          `json:"base_url"`
		APIKeyEnv       string          `json:"api_key_env"`
		MethodAllowlist json.RawMessage `json:"method_allowlist"`
		Enabled         *bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Alias == "" || body.Name == "" || body.BaseURL == "" {
		writeErr(w, http.StatusBadRequest, "alias, name, and base_url required")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	b, err := s.app.CreateMCPBackend(r.Context(), r.PathValue("orgID"), body.Alias, body.Name, body.BaseURL, body.APIKeyEnv, body.MethodAllowlist, enabled)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleUpdateMCPBackend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Alias           *string         `json:"alias"`
		Name            *string         `json:"name"`
		BaseURL         *string         `json:"base_url"`
		APIKeyEnv       *string         `json:"api_key_env"`
		MethodAllowlist json.RawMessage `json:"method_allowlist"`
		Enabled         *bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Alias == nil && body.Name == nil && body.BaseURL == nil && body.APIKeyEnv == nil &&
		body.MethodAllowlist == nil && body.Enabled == nil {
		writeErr(w, http.StatusBadRequest, "at least one field required")
		return
	}
	b, err := s.app.UpdateMCPBackend(r.Context(), r.PathValue("backendID"), body.Alias, body.Name, body.BaseURL, body.APIKeyEnv, body.MethodAllowlist, body.Enabled)
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
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleDeleteMCPBackend(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteMCPBackend(r.Context(), r.PathValue("backendID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestMCPBackend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseURL   string `json:"base_url"`
		APIKeyEnv string `json:"api_key_env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.BaseURL) == "" {
		writeErr(w, http.StatusBadRequest, "base_url required")
		return
	}
	result, err := probeMCP(r.Context(), body.BaseURL, body.APIKeyEnv)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
