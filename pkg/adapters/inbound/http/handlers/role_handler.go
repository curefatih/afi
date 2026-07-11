package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

type RoleHandler struct {
	adminUseCase ports.PlatformAdminUseCase
	authUseCase  ports.AuthUseCase // Combined here to bridge API key mutations
}

func NewRoleHandler(auc ports.PlatformAdminUseCase, auth ports.AuthUseCase) *RoleHandler {
	return &RoleHandler{
		adminUseCase: auc,
		authUseCase:  auth,
	}
}

type CustomRoleRequest struct {
	Name        string                    `json:"name"`
	Scope       string                    `json:"scope"` // "ORGANIZATION" or "PROJECT"
	TargetID    string                    `json:"target_id"`
	Permissions []domain.ActionPermission `json:"permissions"`
}

type RegisterKeyResponse struct {
	APIKey string `json:"api_key"`
}

func (h *RoleHandler) CreateCustomRole(w http.ResponseWriter, r *http.Request) {
	var req CustomRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body json structure")
		return
	}

	roleDomain := &domain.CustomRole{
		Name:        req.Name,
		Scope:       req.Scope,
		TargetID:    req.TargetID,
		Permissions: req.Permissions,
	}

	role, err := h.adminUseCase.CreateCustomRole(r.Context(), roleDomain)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to persist custom policy layout definition")
		return
	}

	h.respondJSON(w, http.StatusCreated, role)
}

func (h *RoleHandler) RegisterProjectKey(w http.ResponseWriter, r *http.Request) {
	// Native Go 1.22 context path value extraction matching our route parameters
	projectID := r.PathValue("project_id")
	if projectID == "" {
		h.respondError(w, http.StatusBadRequest, "Missing path tracking routing element: project_id")
		return
	}

	// Leverage your existing AuthUseCase mapping configuration rule
	rawKey, err := h.authUseCase.IssueAPIKey(r.Context(), domain.APIKeyType("PROJECT"), projectID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Crypto generator validation failure issuing project token")
		return
	}

	h.respondJSON(w, http.StatusOK, RegisterKeyResponse{APIKey: rawKey})
}

func (h *RoleHandler) respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *RoleHandler) respondError(w http.ResponseWriter, code int, msg string) {
	h.respondJSON(w, code, map[string]string{"error": msg})
}
