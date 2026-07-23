package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListRoutes(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListRoutes(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Route{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model           string          `json:"model"`
		ProviderID      string          `json:"provider_id"`
		TargetModel     string          `json:"target_model"`
		Fallbacks       []RouteFallback `json:"fallbacks"`
		Retry           *RetryConfig    `json:"retry"`
		RoutingStrategy string          `json:"routing_strategy"`
		Weight          int             `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Model == "" || body.ProviderID == "" {
		writeErr(w, http.StatusBadRequest, "model and provider_id required")
		return
	}
	if body.TargetModel == "" {
		body.TargetModel = body.Model
	}
	route, err := s.app.CreateRoute(r.Context(), r.PathValue("orgID"), body.Model, body.ProviderID, body.TargetModel, body.Fallbacks, body.Retry, body.RoutingStrategy, body.Weight)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, route)
}

func (s *Server) handleUpdateRoute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model           string          `json:"model"`
		ProviderID      string          `json:"provider_id"`
		TargetModel     string          `json:"target_model"`
		Fallbacks       []RouteFallback `json:"fallbacks"`
		Retry           *RetryConfig    `json:"retry"`
		RoutingStrategy string          `json:"routing_strategy"`
		Weight          int             `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Model == "" || body.ProviderID == "" {
		writeErr(w, http.StatusBadRequest, "model and provider_id required")
		return
	}
	if body.TargetModel == "" {
		body.TargetModel = body.Model
	}
	route, err := s.app.UpdateRoute(r.Context(), r.PathValue("routeID"), body.Model, body.ProviderID, body.TargetModel, body.Fallbacks, body.Retry, body.RoutingStrategy, body.Weight)
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
	writeJSON(w, http.StatusOK, route)
}

func (s *Server) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteRoute(r.Context(), r.PathValue("routeID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
