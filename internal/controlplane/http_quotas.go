package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListQuotas(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListQuotas(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Quota{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateQuota(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ScopeType  string `json:"scope_type"`
		ScopeID    string `json:"scope_id"`
		Metric     string `json:"metric"`
		LimitValue int64  `json:"limit_value"`
		Window     string `json:"window"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ScopeType == "" || body.ScopeID == "" || body.Metric == "" {
		writeErr(w, http.StatusBadRequest, "scope_type, scope_id, metric required")
		return
	}
	if body.LimitValue < 0 {
		writeErr(w, http.StatusBadRequest, "limit_value must be >= 0")
		return
	}
	q, err := s.app.CreateQuota(r.Context(), r.PathValue("orgID"), body.ScopeType, body.ScopeID, body.Metric, body.LimitValue, body.Window)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, q)
}

func (s *Server) handleUpdateQuota(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LimitValue int64 `json:"limit_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LimitValue < 0 {
		writeErr(w, http.StatusBadRequest, "limit_value required (>= 0)")
		return
	}
	q, err := s.app.UpdateQuota(r.Context(), r.PathValue("quotaID"), body.LimitValue)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) handleDeleteQuota(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteQuota(r.Context(), r.PathValue("quotaID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
