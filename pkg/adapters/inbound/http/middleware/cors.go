package middleware

import (
	"net/http"
	"strings"
)

// CORS configurations structural rules wrapper
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// DefaultCORSConfig provides a safe baseline development schema configuration layout
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept", "X-Requested-With"},
		AllowCredentials: true,
	}
}

// CORS wraps an http.Handler to apply browser validation security headers completely framework-free
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	allowedMethods := strings.Join(config.AllowedMethods, ", ")
	allowedHeaders := strings.Join(config.AllowedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Not a CORS request, pass straight down the execution pipe
				next.ServeHTTP(w, r)
				return
			}

			// Check origin rules alignment match conditions
			isAllowed := false
			for _, allowed := range config.AllowedOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				http.Error(w, "CORS Policy Origin Disallowed", http.StatusForbidden)
				return
			}

			// Assign common handshake security requirements definitions
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
