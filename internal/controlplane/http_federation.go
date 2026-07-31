package controlplane

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/federationclient"
	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/usage"
)

func federationTokenFrom(r *http.Request) string {
	if t := strings.TrimSpace(r.Header.Get(federationclient.TokenHeader)); t != "" {
		return t
	}
	return ""
}

func (s *Server) handleListFederationPeers(w http.ResponseWriter, r *http.Request) {
	items, err := s.app.ListFederationPeers(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []federation.ControlPlanePeer{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleRegisterFederationPeer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		RegionID string `json:"region_id"`
		BaseURL  string `json:"base_url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := s.app.RegisterFederationPeer(r.Context(), body.Name, body.RegionID, body.BaseURL)
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

func (s *Server) handleGetFederationPeer(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.GetFederationPeer(r.Context(), r.PathValue("peerID"))
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

func (s *Server) handleUpdateFederationPeer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    *string `json:"name"`
		BaseURL *string `json:"base_url"`
		Status  *string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	name, baseURL, status := "", "", ""
	if body.Name != nil {
		name = *body.Name
	}
	if body.BaseURL != nil {
		baseURL = *body.BaseURL
	}
	if body.Status != nil {
		status = *body.Status
	}
	p, err := s.app.UpdateFederationPeer(r.Context(), r.PathValue("peerID"), name, baseURL, status)
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

func (s *Server) handleRotateFederationPeerToken(w http.ResponseWriter, r *http.Request) {
	out, err := s.app.RotateFederationPeerJoinToken(r.Context(), r.PathValue("peerID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleFederationPeerJoin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JoinToken string `json:"join_token"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
	token := strings.TrimSpace(body.JoinToken)
	if token == "" {
		token = federationTokenFrom(r)
	}
	p, err := s.app.JoinFederationPeer(r.Context(), token)
	if errors.Is(err, kernel.ErrUnauthorized) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleFederationRegionExport(w http.ResponseWriter, r *http.Request) {
	token := federationTokenFrom(r)
	peer, err := s.app.AuthenticateFederationPeerToken(r.Context(), token)
	if errors.Is(err, kernel.ErrUnauthorized) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	slug := strings.TrimSpace(r.PathValue("slug"))
	reg, err := s.app.GetRegion(r.Context(), peer.RegionID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if reg.Slug != slug {
		writeErr(w, http.StatusForbidden, "peer not scoped to this region")
		return
	}
	var since int64
	if q := r.URL.Query().Get("since"); q != "" {
		since, _ = strconv.ParseInt(q, 10, 64)
	}
	prefix := "snapshots"
	if s.cfg != nil && s.cfg.Gateway.SnapshotS3.Prefix != "" {
		prefix = s.cfg.Gateway.SnapshotS3.Prefix
	}
	exp, err := s.app.ExportFederationRegion(r.Context(), slug, since, prefix)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.app.RecordFederationPeerSync(r.Context(), peer.ID, exp.Revision, "")
	writeJSON(w, http.StatusOK, exp)
}

// handleFederationUsageReports serves local usage_events to the hub (regional CP only).
// Auth: X-AFI-Federation-Token must match this regional CP's join token.
func (s *Server) handleFederationUsageReports(w http.ResponseWriter, r *http.Request) {
	if s.cfg == nil || s.cfg.Federation.Mode != "regional" {
		writeErr(w, http.StatusForbidden, "usage reports are served by regional control planes")
		return
	}
	raw := federationTokenFrom(r)
	if raw == "" || strings.TrimSpace(s.cfg.Federation.JoinToken) == "" {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if identity.HashOpaqueToken(raw) != identity.HashOpaqueToken(s.cfg.Federation.JoinToken) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var since *time.Time
	if q := strings.TrimSpace(r.URL.Query().Get("since")); q != "" {
		t, err := time.Parse(time.RFC3339, q)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "since must be RFC3339")
			return
		}
		t = t.UTC()
		since = &t
	}
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			limit = n
		}
	}
	items, err := s.app.ListFederationUsageReports(r.Context(), since, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []usage.Record{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": items})
}

// handlePullPeerUsageReports fetches usage reports from a regional peer on demand (hub).
// Does not persist reports on the hub. Requires X-AFI-Federation-Token (peer join token).
func (s *Server) handlePullPeerUsageReports(w http.ResponseWriter, r *http.Request) {
	peerID := r.PathValue("peerID")
	peer, err := s.app.GetFederationPeer(r.Context(), peerID)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	token := federationTokenFrom(r)
	authed, err := s.app.AuthenticateFederationPeerToken(r.Context(), token)
	if errors.Is(err, kernel.ErrUnauthorized) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if authed.ID != peer.ID {
		writeErr(w, http.StatusForbidden, "token not scoped to this peer")
		return
	}
	if strings.TrimSpace(peer.BaseURL) == "" {
		writeErr(w, http.StatusBadRequest, "peer base_url required to pull usage reports")
		return
	}
	var since *time.Time
	if q := strings.TrimSpace(r.URL.Query().Get("since")); q != "" {
		t, err := time.Parse(time.RFC3339, q)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "since must be RFC3339")
			return
		}
		t = t.UTC()
		since = &t
	}
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			limit = n
		}
	}
	client := federationclient.New(peer.BaseURL, token)
	out, err := client.UsageReports(r.Context(), since, limit)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	for _, rep := range out.Reports {
		region := ""
		if rep.Tags != nil {
			region = rep.Tags["region"]
		}
		if s.Metrics != nil {
			s.Metrics.RecordSpokeUsage(r.Context(), region, rep.Status, usage.NormalizeModality(rep.Modality), rep.PromptTokens, rep.CompletionTokens)
		}
		if s.log != nil {
			s.log.Info("pulled spoke usage report",
				"peer_id", peer.ID,
				"region", region,
				"org", rep.OrganizationID,
				"model", rep.Model,
				"status", rep.Status,
				"prompt_tokens", rep.PromptTokens,
				"completion_tokens", rep.CompletionTokens,
			)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
