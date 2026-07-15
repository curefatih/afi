package handlers

import (
	"net/http"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
	"github.com/curefatih/afi/pkg/adapters/inbound/http/middleware"
)

func RegisterPlatformRoutes(
	mux *http.ServeMux,
	tokenSvc ports.PlatformTokenService,
	userHandler *UserHandler,
	roleHandler *RoleHandler,
) {

	protectAuth := func(next http.Handler) http.Handler {
		return middleware.RequireAuth(tokenSvc, domain.PermOrgUserRead)(next)
	}

	protectPermission := func(required domain.ActionPermission, next http.Handler) http.Handler {
		return middleware.RequirePermission(tokenSvc, required)(next)
	}

	mux.HandleFunc("POST /api/v1/platform/auth/login", userHandler.Login)
	mux.Handle("GET /api/v1/platform/auth/me",
		protectAuth(protectPermission(domain.PermOrgUserRead, http.HandlerFunc(userHandler.GetMe))),
	)

	mux.Handle("GET /api/v1/platform/organizations", protectAuth(http.HandlerFunc(userHandler.GetUserOrganizations)))

	mux.Handle("POST /api/v1/platform/organizations/{org_id}/users", protectAuth(protectPermission(domain.PermOrgUserWrite, http.HandlerFunc(userHandler.CreateUser))))
	mux.Handle("POST /api/v1/platform/organizations/{org_id}/roles/custom", protectAuth(protectPermission(domain.PermOrgRoleWrite, http.HandlerFunc(roleHandler.CreateCustomRole))))
	mux.Handle("GET /api/v1/platform/organizations/{org_id}/projects",
		protectAuth(protectPermission(domain.PermOrgUserRead, http.HandlerFunc(userHandler.GetUserOrganizationProjects))),
	)
	mux.Handle("GET /api/v1/platform/organizations/{org_id}/teams",
		protectAuth(protectPermission(domain.PermOrgUserRead, http.HandlerFunc(userHandler.GetUserOrganizationProjects))),
	)

	mux.Handle("POST /api/v1/platform/organizations/{org_id}/projects/{project_id}/keys", protectAuth(protectPermission(domain.PermProjectKeyWrite, http.HandlerFunc(roleHandler.RegisterProjectKey))))

}
