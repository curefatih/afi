package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

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
