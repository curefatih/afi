package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/curefatih/afi/internal/kernel"
)

func (s *Server) handleListSigningKeys(w http.ResponseWriter, r *http.Request) {
	list, err := s.app.ListSigningKeys(r.Context(), r.PathValue("orgID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []SigningKey{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateSigningKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		KeyID         string `json:"key_id"`
		ProjectID     string `json:"project_id"`
		EnvironmentID string `json:"environment_id"`
		Name          string `json:"name"`
		Algorithm     string `json:"algorithm"`
		PublicKeyPEM  string `json:"public_key_pem"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	k, err := s.app.CreateSigningKey(r.Context(), r.PathValue("orgID"), body.KeyID, body.ProjectID, body.EnvironmentID, body.Name, body.Algorithm, body.PublicKeyPEM)
	if errors.Is(err, kernel.ErrInvalidRequest) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (s *Server) handleUpdateSigningKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	k, err := s.app.UpdateSigningKey(r.Context(), r.PathValue("signingKeyID"), body.Name, body.Status)
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
	writeJSON(w, http.StatusOK, k)
}

func (s *Server) handleRotateSigningKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PublicKeyPEM string `json:"public_key_pem"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	k, err := s.app.RotateSigningKey(r.Context(), r.PathValue("signingKeyID"), body.PublicKeyPEM)
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
	writeJSON(w, http.StatusOK, k)
}

func (s *Server) handleDeleteSigningKey(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteSigningKey(r.Context(), r.PathValue("signingKeyID")); errors.Is(err, kernel.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
