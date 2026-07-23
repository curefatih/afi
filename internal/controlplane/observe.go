package controlplane

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type cpStatusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *cpStatusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *cpStatusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *cpStatusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func (s *Server) withCPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if s.Metrics == nil || path == "/healthz" || path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &cpStatusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		route := cpRouteTemplate(path)
		s.Metrics.Record(r.Context(), route, statusClass(rec.code), time.Since(start).Seconds())
	})
}

func statusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}

// cpRouteTemplate collapses path IDs to keep metric cardinality low.
func cpRouteTemplate(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if looksLikeID(p) {
			parts[i] = "{" + idLabel(parts, i) + "}"
		}
	}
	out := strings.Join(parts, "/")
	if out == "" {
		return "/"
	}
	return out
}

func looksLikeID(s string) bool {
	if len(s) >= 20 {
		return true
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil && len(s) > 3 {
		return true
	}
	// UUID-ish
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return true
	}
	return false
}

func idLabel(parts []string, i int) string {
	if i > 0 {
		switch parts[i-1] {
		case "organizations":
			return "orgID"
		case "projects":
			return "projectID"
		case "teams":
			return "teamID"
		case "providers":
			return "providerID"
		case "routes":
			return "routeID"
		case "quotas":
			return "quotaID"
		case "policies":
			return "policyID"
		case "keys":
			return "keyID"
		case "members":
			return "userID"
		case "credentials":
			return "credentialID"
		case "mcp-backends":
			return "backendID"
		case "a2a-agents":
			return "agentID"
		case "wasm-hooks":
			return "hookID"
		case "invites":
			return "inviteID"
		case "sso":
			return "provider"
		}
	}
	return "id"
}
