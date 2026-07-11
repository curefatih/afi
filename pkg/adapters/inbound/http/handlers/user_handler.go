package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/curefatih/afi/internal/ports"
)

type UserHandler struct {
	adminUseCase ports.PlatformAdminUseCase
}

func NewUserHandler(auc ports.PlatformAdminUseCase) *UserHandler {
	return &UserHandler{adminUseCase: auc}
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
	token, err := h.adminUseCase.LoginPlatformWithEmailAndPassword(r.Context(), req.Email, req.Password)
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

	user, err := h.adminUseCase.RegisterAdminUser(r.Context(), req.Email, req.Password)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to register new platform administrator")
		return
	}

	h.respondJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *UserHandler) respondError(w http.ResponseWriter, code int, msg string) {
	h.respondJSON(w, code, map[string]string{"error": msg})
}
