package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListWasmHooks(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListWasmHooks(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []WasmHook{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateWasmHook(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string          `json:"name"`
		Phase     string          `json:"phase"`
		ModuleURI string          `json:"module_uri"`
		Digest    string          `json:"digest"`
		Enabled   *bool           `json:"enabled"`
		Priority  *int            `json:"priority"`
		Config    json.RawMessage `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Phase == "" || body.ModuleURI == "" {
		writeErr(w, http.StatusBadRequest, "name, phase, and module_uri required")
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
	h, err := s.app.CreateWasmHook(r.Context(), r.PathValue("orgID"), body.Name, body.Phase, body.ModuleURI, body.Digest, enabled, priority, body.Config)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, h)
}

func (s *Server) handleUpdateWasmHook(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      *string         `json:"name"`
		Phase     *string         `json:"phase"`
		ModuleURI *string         `json:"module_uri"`
		Digest    *string         `json:"digest"`
		Enabled   *bool           `json:"enabled"`
		Priority  *int            `json:"priority"`
		Config    json.RawMessage `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == nil && body.Phase == nil && body.ModuleURI == nil && body.Digest == nil &&
		body.Enabled == nil && body.Priority == nil && body.Config == nil {
		writeErr(w, http.StatusBadRequest, "at least one field required")
		return
	}
	h, err := s.app.UpdateWasmHook(r.Context(), r.PathValue("hookID"), body.Name, body.Phase, body.ModuleURI, body.Digest, body.Enabled, body.Priority, body.Config)
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
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleDeleteWasmHook(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteWasmHook(r.Context(), r.PathValue("hookID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
