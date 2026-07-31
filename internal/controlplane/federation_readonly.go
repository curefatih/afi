package controlplane

import (
	"net/http"
	"strings"
)

// federationRegionalReadOnly enforces data-locality management rules on a regional CP:
// config is pulled from the hub; mutating platform APIs and hub-only federation/internal
// endpoints are rejected. Auth and read APIs remain available for local visibility.
func (s *Server) federationRegionalReadOnly(next http.Handler) http.Handler {
	if s == nil || s.cfg == nil || s.cfg.Federation.Mode != "regional" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			if strings.HasPrefix(path, "/internal/v1/federation/") {
				// Regional CP may serve usage reports to the hub; config export stays hub-only.
				if strings.HasSuffix(path, "/usage-reports") {
					next.ServeHTTP(w, r)
					return
				}
				writeErr(w, http.StatusForbidden, "federation export is hub-only; regional CP pulls from the hub")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		// Auth (login, password reset, invite accept, SSO) must work locally.
		if strings.HasPrefix(path, "/api/v1/platform/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		writeErr(w, http.StatusForbidden, "regional control plane is read-only for management; change config on the hub")
	})
}
