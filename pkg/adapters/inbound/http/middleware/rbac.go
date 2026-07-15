package middleware

import (
	"context"
	"net/http"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

func RequirePermission(
	tokenSvc ports.PlatformTokenService,
	required domain.ActionPermission,
) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok || userID == "" {
				respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
				return
			}

			orgID := r.PathValue("org_id")
			projectID := r.PathValue("project_id")

			permissions, err := tokenSvc.GetUserPermissions(r.Context(), userID, orgID, projectID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to fetch user permissions")
				return
			}

			hasAccess := false
			for _, permission := range permissions {
				if permission == required {
					hasAccess = true
					break
				}
			}

			if !hasAccess {
				respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
