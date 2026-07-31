package controlplane

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/curefatih/afi/internal/adapters/federationclient"
	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/kernel"
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
