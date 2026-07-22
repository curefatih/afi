package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleGetOrgDefaultRetry(w http.ResponseWriter, r *http.Request) {
	retry, err := s.app.GetOrgDefaultRetry(r.Context(), r.PathValue("orgID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"retry": retry})
}

func (s *Server) handleUpdateOrgDefaultRetry(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Retry *RetryConfig `json:"retry"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	retry, err := s.app.SetOrgDefaultRetry(r.Context(), r.PathValue("orgID"), body.Retry)
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
	writeJSON(w, http.StatusOK, map[string]any{"retry": retry})
}
