package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListCredentials(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Credential{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		ProviderType string `json:"provider_type"`
		StorageKind  string `json:"storage_kind"`
		SecretRef    string `json:"secret_ref"`
		SecretValue  string `json:"secret_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.ProviderType == "" || body.StorageKind == "" {
		writeErr(w, http.StatusBadRequest, "name, provider_type, and storage_kind required")
		return
	}
	c, err := s.app.CreateCredential(r.Context(), r.PathValue("orgID"), body.Name, body.ProviderType, body.StorageKind, body.SecretRef, body.SecretValue)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" && body.Status == "" {
		writeErr(w, http.StatusBadRequest, "name or status required")
		return
	}
	c, err := s.app.UpdateCredential(r.Context(), r.PathValue("credentialID"), body.Name, body.Status)
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
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleRotateCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SecretRef   string `json:"secret_ref"`
		SecretValue string `json:"secret_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	c, err := s.app.RotateCredential(r.Context(), r.PathValue("credentialID"), body.SecretRef, body.SecretValue)
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
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	err := s.app.DeleteCredential(r.Context(), r.PathValue("credentialID"))
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, kernel.ErrConflict) {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListCredentialAssignments(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListCredentialAssignments(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []CredentialAssignment{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAssignCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CredentialID string `json:"credential_id"`
		ScopeType    string `json:"scope_type"`
		ScopeID      string `json:"scope_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CredentialID == "" || body.ScopeType == "" || body.ScopeID == "" {
		writeErr(w, http.StatusBadRequest, "credential_id, scope_type, and scope_id required")
		return
	}
	orgID := r.PathValue("orgID")
	credOrg, err := s.members.GetCredentialOrgID(r.Context(), body.CredentialID)
	if errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if credOrg != orgID {
		writeErr(w, http.StatusBadRequest, "credential does not belong to organization")
		return
	}
	claims := claimsFrom(r.Context())
	a, err := s.app.AssignCredential(r.Context(), body.CredentialID, body.ScopeType, body.ScopeID, claims.UserID)
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
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleDeleteCredentialAssignment(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteCredentialAssignment(r.Context(), r.PathValue("assignmentID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
