package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

type contextKey string

const UserIDKey contextKey = "userID"
const OrgIDKey contextKey = "orgID"

func RequireAuth(
	tokenSvc ports.PlatformTokenService,
	required domain.ActionPermission,
) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
				return
			}

			tokenStr := extractToken(authHeader)
			userID, err := tokenSvc.ValidateToken(r.Context(), tokenStr)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, message)))
}

func extractToken(authHeader string) string {
	return strings.TrimPrefix(authHeader, "Bearer ")
}
