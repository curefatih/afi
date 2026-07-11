package http

import (
	"net/http"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/handlers"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/middleware"
)

func RegisterPlatformRoutes(
	mux *http.ServeMux,
	tokenSvc ports.PlatformTokenService,
	userHandler *handlers.UserHandler,
	roleHandler *handlers.RoleHandler,
) {

	protect := func(required domain.ActionPermission, next http.Handler) http.Handler {
		return middleware.RequirePermission(tokenSvc, required)(next)
	}

	mux.HandleFunc("POST /api/v1/platform/auth/login", userHandler.Login)

	mux.Handle("POST /api/v1/platform/organizations/{org_id}/users", protect(domain.PermOrgUserWrite, http.HandlerFunc(userHandler.CreateUser)))
	mux.Handle("POST /api/v1/platform/organizations/{org_id}/roles/custom", protect(domain.PermOrgRoleWrite, http.HandlerFunc(roleHandler.CreateCustomRole)))

	mux.Handle("POST /api/v1/platform/organizations/{org_id}/projects/{project_id}/keys", protect(domain.PermProjectKeyWrite, http.HandlerFunc(roleHandler.RegisterProjectKey)))

}
