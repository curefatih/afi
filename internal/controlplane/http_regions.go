package controlplane

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
)

func (s *Server) handleListRegions(w http.ResponseWriter, r *http.Request) {
	items, err := s.app.ListRegions(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []regions.Region{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateRegion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	reg, err := s.app.CreateRegion(r.Context(), body.Slug, body.Name)
	if err != nil {
		if errors.Is(err, kernel.ErrInvalidRequest) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reg)
}

func (s *Server) handleGetRegion(w http.ResponseWriter, r *http.Request) {
	reg, err := s.app.GetRegion(r.Context(), r.PathValue("regionID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reg)
}

func (s *Server) handleUpdateRegion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	name, status := "", ""
	if body.Name != nil {
		name = *body.Name
	}
	if body.Status != nil {
		status = *body.Status
	}
	reg, err := s.app.UpdateRegion(r.Context(), r.PathValue("regionID"), name, status)
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
	writeJSON(w, http.StatusOK, reg)
}

func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	items, err := s.app.ListDeployments(r.Context(), r.PathValue("regionID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []regions.GatewayDeployment{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleRegisterDeployment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string `json:"name"`
		PublicBaseURL string `json:"public_base_url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := s.app.RegisterDeployment(r.Context(), r.PathValue("regionID"), body.Name, body.PublicBaseURL)
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
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	d, err := s.app.GetDeployment(r.Context(), r.PathValue("deploymentID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if d.RegionID != r.PathValue("regionID") {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleRotateDeploymentJoinToken(w http.ResponseWriter, r *http.Request) {
	d, err := s.app.GetDeployment(r.Context(), r.PathValue("deploymentID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if d.RegionID != r.PathValue("regionID") {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	out, err := s.app.RotateDeploymentJoinToken(r.Context(), r.PathValue("deploymentID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDeploymentHeartbeat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SnapshotVersion int64  `json:"snapshot_version"`
		Build           string `json:"build"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
	token := deploymentJoinTokenFrom(r)
	d, err := s.app.RecordDeploymentHeartbeat(r.Context(), r.PathValue("deploymentID"), token, body.SnapshotVersion, body.Build)
	if errors.Is(err, kernel.ErrUnauthorized) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleDeploymentUsageIngest(w http.ResponseWriter, r *http.Request) {
	token := deploymentJoinTokenFrom(r)
	d, err := s.app.AuthenticateDeploymentJoinToken(r.Context(), token)
	if errors.Is(err, kernel.ErrUnauthorized) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if d.ID != r.PathValue("deploymentID") {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || len(payload) == 0 {
		writeErr(w, http.StatusBadRequest, "payload required")
		return
	}
	if s.usageEnqueuer == nil {
		writeErr(w, http.StatusServiceUnavailable, "usage ingest unavailable")
		return
	}
	if err := s.usageEnqueuer.Enqueue(r.Context(), payload); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDeploymentJoin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JoinToken string `json:"join_token"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	d, err := s.app.AuthenticateDeploymentJoinToken(r.Context(), body.JoinToken)
	if errors.Is(err, kernel.ErrUnauthorized) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deployment_id": d.ID,
		"region_id":     d.RegionID,
		"name":          d.Name,
		"status":        d.Status,
	})
}

func (s *Server) handleListRegionMemberships(w http.ResponseWriter, r *http.Request) {
	items, err := s.app.ListRegionMemberships(r.Context(), r.PathValue("regionID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []regions.OrgRegionMembership{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleBindOrgToRegion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrganizationID string `json:"organization_id"`
		Status         string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	m, err := s.app.BindOrgToRegion(r.Context(), r.PathValue("regionID"), body.OrganizationID, body.Status)
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
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleUnbindOrgFromRegion(w http.ResponseWriter, r *http.Request) {
	err := s.app.UnbindOrgFromRegion(r.Context(), r.PathValue("regionID"), r.PathValue("orgID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetRegionOverlay(w http.ResponseWriter, r *http.Request) {
	o, err := s.app.GetRegionOverlay(r.Context(), r.PathValue("regionID"), r.PathValue("orgID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) handlePutRegionOverlay(w http.ResponseWriter, r *http.Request) {
	var body regions.OverlayPayload
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	o, err := s.app.PutRegionOverlay(r.Context(), r.PathValue("regionID"), r.PathValue("orgID"), body)
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
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) handleDeleteRegionOverlay(w http.ResponseWriter, r *http.Request) {
	err := s.app.DeleteRegionOverlay(r.Context(), r.PathValue("regionID"), r.PathValue("orgID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deploymentJoinTokenFrom(r *http.Request) string {
	if t := strings.TrimSpace(r.Header.Get("X-AFI-Deployment-Token")); t != "" {
		return t
	}
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return ""
}

func (s *Server) requirePlatformAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		if claims == nil || !identity.IsPlatformAdmin(claims.Role) {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}
