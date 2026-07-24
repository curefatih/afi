package controlplane

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.app.ListAPIKeys(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = []APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleListOrgKeys(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgID")
	claims := claimsFrom(r.Context())
	admin, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	keys, err := s.app.ListVisibleOrgAPIKeys(r.Context(), orgID, claims.UserID, admin)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = []APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleCreateOrgKey(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgID")
	claims := claimsFrom(r.Context())
	var body struct {
		Name          string `json:"name"`
		Key           string `json:"key"`
		Kind          string `json:"kind"`
		ProjectID     string `json:"project_id"`
		EnvironmentID string `json:"environment_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" {
		body.Name = "API Key"
	}
	if body.Key == "" {
		body.Key = "sk-" + randomHex(24)
	}
	if body.Kind == "" {
		body.Kind = snapshot.KeyKindPersonal
	}

	ownerUserID := ""
	switch body.Kind {
	case snapshot.KeyKindPersonal:
		if body.ProjectID != "" {
			writeErr(w, http.StatusBadRequest, "personal keys cannot have a project")
			return
		}
		if body.EnvironmentID != "" {
			writeErr(w, http.StatusBadRequest, "personal keys cannot have an environment")
			return
		}
		ownerUserID = claims.UserID
	case snapshot.KeyKindServiceAccount:
		admin, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, orgID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !admin {
			writeErr(w, http.StatusForbidden, "only org admins can create service account keys")
			return
		}
	default:
		writeErr(w, http.StatusBadRequest, "kind must be personal or service_account")
		return
	}

	k, err := s.app.CreateAPIKey(r.Context(), orgID, body.Kind, ownerUserID, body.ProjectID, body.EnvironmentID, body.Name, body.Key)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	keyID := r.PathValue("keyID")
	k, err := s.api.GetAPIKey(r.Context(), keyID)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok, err := s.members.IsOrgMember(r.Context(), claims.UserID, k.OrganizationID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	admin, err := s.members.IsOrgAdmin(r.Context(), claims.UserID, k.OrganizationID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !admin && !(k.Kind == snapshot.KeyKindPersonal && k.OwnerUserID == claims.UserID) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := s.app.DeleteAPIKey(r.Context(), keyID); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string `json:"name"`
		Key           string `json:"key"`
		EnvironmentID string `json:"environment_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" {
		body.Name = "API Key"
	}
	if body.Key == "" {
		body.Key = "sk-" + randomHex(24)
	}

	orgID, err := s.api.GetProjectOrgID(r.Context(), r.PathValue("projectID"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}

	k, err := s.app.CreateAPIKey(r.Context(), orgID, snapshot.KeyKindServiceAccount, "", r.PathValue("projectID"), body.EnvironmentID, body.Name, body.Key)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, k)
}
