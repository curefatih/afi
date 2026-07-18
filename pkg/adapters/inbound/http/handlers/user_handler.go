package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/curefatih/afi/internal/ports"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/middleware"
)

type UserHandler struct {
	userUseCase ports.PlatformUserUseCase
}

func NewUserHandler(auc ports.PlatformUserUseCase) *UserHandler {
	return &UserHandler{userUseCase: auc}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body json structure")
		return
	}

	if req.Email == "" || req.Password == "" {
		h.respondError(w, http.StatusBadRequest, "Email and password are required tokens")
		return
	}

	// Route down to core authentication logic
	token, err := h.userUseCase.LoginPlatformWithEmailAndPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "Invalid platform administration credentials")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"token": token.Token})
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body json structure")
		return
	}

	if req.Email == "" || req.Password == "" {
		h.respondError(w, http.StatusBadRequest, "Email and password parameters must be supplied")
		return
	}

	user, err := h.userUseCase.RegisterPlatformUser(r.Context(), req.Email, req.Password)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to register new platform user")
		return
	}

	h.respondJSON(w, http.StatusCreated, user)
}

// GET /admin/v1/organizations/{org_id}/auth/me
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure UserID string injected by your RequireAuth middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unidentified session context")
		return
	}

	// 2. Fetch the profile via your inbound port service layer
	user, err := h.userUseCase.GetProfileByID(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "User profile record not found")
		return
	}

	// 3. Return the clean domain representation out to your dashboard UI
	h.respondJSON(w, http.StatusOK, user)
}

func (h *UserHandler) GetUserOrganizations(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure UserID string injected by your RequireAuth middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unidentified session context")
		return
	}

	// 2. Fetch the organizations via your inbound port service layer
	orgs, err := h.userUseCase.GetUserOrganizations(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user organizations")
		return
	}

	// 3. Return the clean domain representation out to your dashboard UI
	h.respondJSON(w, http.StatusOK, orgs)
}

// GET /platform/v1/organizations/{org_id}/projects
func (h *UserHandler) GetUserOrganizationProjects(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure UserID string injected by your RequireAuth middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unidentified session context")
		return
	}

	orgID := r.PathValue("org_id")
	if orgID == "" {
		h.respondError(w, http.StatusInternalServerError, "Failed to read organization id from path: /api/v1/platform/organizations/{org_id}/projects")
	}

	// 2. Fetch the projects via your inbound port service layer
	orgs, err := h.userUseCase.GetUserOrganizationProjects(r.Context(), userID, orgID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user projects")
		return
	}

	// 3. Return the clean domain representation out to your dashboard UI
	h.respondJSON(w, http.StatusOK, orgs)
}

// GET /platform/v1/organizations/{org_id}/teams
func (h *UserHandler) GetUserOrganizationTeams(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure UserID string injected by your RequireAuth middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unidentified session context")
		return
	}

	orgID := r.PathValue("org_id")
	if orgID == "" {
		h.respondError(w, http.StatusInternalServerError, "Failed to read organization id from path: /api/v1/platform/organizations/{org_id}/teams")
	}

	// 2. Fetch the team via your inbound port service layer
	teams, err := h.userUseCase.GetUserOrganizationTeams(r.Context(), userID, orgID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user teams")
		return
	}

	// 3. Return the clean domain representation out to your dashboard UI
	h.respondJSON(w, http.StatusOK, teams)
}

// GET /platform/v1/teams/{team_id}
func (h *UserHandler) GetUserTeam(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure UserID string injected by your RequireAuth middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unidentified session context")
		return
	}

	teamID := r.PathValue("team_id")
	if teamID == "" {
		h.respondError(w, http.StatusInternalServerError, "Failed to read organization id from path: /api/v1/platform/v1/teams/{team_id}")
	}

	// 2. Fetch the team via your inbound port service layer
	teams, err := h.userUseCase.GetUserTeam(r.Context(), userID, teamID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user teams")
		return
	}

	// 3. Return the clean domain representation out to your dashboard UI
	h.respondJSON(w, http.StatusOK, teams)
}

func (h *UserHandler) GetUserTeamMembers(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure UserID string injected by your RequireAuth middleware
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unidentified session context")
		return
	}

	teamID := r.PathValue("team_id")
	if teamID == "" {
		h.respondError(w, http.StatusInternalServerError, "Failed to read organization id from path: /api/v1/platform/v1/teams/{team_id}/members")
	}

	// 2. Fetch the team via your inbound port service layer
	members, err := h.userUseCase.GetUserTeamMembers(r.Context(), userID, teamID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user team members")
		return
	}

	// 3. Return the clean domain representation out to your dashboard UI
	h.respondJSON(w, http.StatusOK, members)
}

func (h *UserHandler) respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *UserHandler) respondError(w http.ResponseWriter, code int, msg string) {
	h.respondJSON(w, code, map[string]string{"error": msg})
}
