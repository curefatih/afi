package dialect

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

// WriteUpstreamError remaps an upstream HTTP error body into the client dialect
// and writes it. Status is preserved; Content-Type is always application/json.
func WriteUpstreamError(w http.ResponseWriter, d ir.Dialect, status int, body []byte) {
	msg, typ := ParseErrorBody(body)
	if msg == "" {
		msg = http.StatusText(status)
		if msg == "" {
			msg = "upstream error"
		}
	}
	if typ == "" {
		typ = defaultErrorType(status)
	}
	WriteError(w, d, status, msg, typ)
}

// ParseErrorBody extracts a human message and machine type from common upstream
// error JSON shapes (OpenAI, Anthropic, Gemini, Bedrock). Unknown bodies fall
// back to the trimmed raw string when it looks like plain text.
func ParseErrorBody(body []byte) (message, typ string) {
	body = bytesTrimSpace(body)
	if len(body) == 0 {
		return "", ""
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		s := string(body)
		if looksLikePlainText(s) {
			return truncate(s, 512), ""
		}
		return "", ""
	}

	// Anthropic: {"type":"error","error":{"type":"...","message":"..."}}
	if t, _ := raw["type"].(string); t == "error" {
		if errObj, ok := raw["error"].(map[string]any); ok {
			msg, _ := errObj["message"].(string)
			et, _ := errObj["type"].(string)
			return strings.TrimSpace(msg), et
		}
	}

	// OpenAI / Gemini: {"error":{"message":"...","type":"..."}} or {"error":"..."}
	if errVal, ok := raw["error"]; ok {
		switch e := errVal.(type) {
		case string:
			return strings.TrimSpace(e), ""
		case map[string]any:
			msg, _ := e["message"].(string)
			et, _ := e["type"].(string)
			if et == "" {
				et, _ = e["status"].(string) // Gemini sometimes uses status
			}
			if et == "" {
				et, _ = e["code"].(string)
			}
			return strings.TrimSpace(msg), et
		}
	}

	// Bedrock / flat: {"message":"...","code":"..."}
	if msg, ok := raw["message"].(string); ok && strings.TrimSpace(msg) != "" {
		et, _ := raw["code"].(string)
		if et == "" {
			et, _ = raw["type"].(string)
		}
		if et == "" {
			et, _ = raw["__type"].(string)
		}
		return strings.TrimSpace(msg), et
	}
	return "", ""
}

func defaultErrorType(status int) string {
	switch {
	case status == http.StatusTooManyRequests:
		return "rate_limit_error"
	case status == http.StatusUnauthorized:
		return "authentication_error"
	case status == http.StatusForbidden:
		return "permission_error"
	case status == http.StatusNotFound:
		return "not_found_error"
	case status >= 500:
		return "api_error"
	default:
		return "invalid_request_error"
	}
}

// normalizeErrorType maps vendor-specific codes onto the client's dialect vocabulary.
func normalizeErrorType(d ir.Dialect, typ string) string {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return typ
	}
	switch d {
	case ir.DialectAnthropic:
		switch strings.ToLower(typ) {
		case "server_error", "internal_server_error":
			return "api_error"
		case "insufficient_quota":
			return "rate_limit_error"
		case "policy_violation", "hook_denied":
			return "permission_error"
		default:
			return typ
		}
	case ir.DialectGemini:
		switch strings.ToLower(typ) {
		case "invalid_request_error":
			return "INVALID_ARGUMENT"
		case "authentication_error":
			return "UNAUTHENTICATED"
		case "permission_error", "policy_violation", "hook_denied":
			return "PERMISSION_DENIED"
		case "rate_limit_error", "insufficient_quota":
			return "RESOURCE_EXHAUSTED"
		case "not_found_error":
			return "NOT_FOUND"
		case "server_error", "api_error", "overloaded_error":
			return "INTERNAL"
		default:
			return strings.ToUpper(typ)
		}
	default: // OpenAI
		switch strings.ToLower(typ) {
		case "api_error", "overloaded_error":
			return "server_error"
		case "permission_error":
			// Keep Anthropic-style permission_error as-is; OpenAI also uses it.
			return typ
		default:
			return typ
		}
	}
}

func geminiErrorStatus(status int, typ string) string {
	upper := strings.ToUpper(strings.TrimSpace(typ))
	switch upper {
	case "INVALID_ARGUMENT", "UNAUTHENTICATED", "PERMISSION_DENIED",
		"RESOURCE_EXHAUSTED", "NOT_FOUND", "INTERNAL", "UNAVAILABLE":
		return upper
	}
	switch {
	case status == http.StatusBadRequest:
		return "INVALID_ARGUMENT"
	case status == http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case status == http.StatusForbidden:
		return "PERMISSION_DENIED"
	case status == http.StatusNotFound:
		return "NOT_FOUND"
	case status == http.StatusTooManyRequests:
		return "RESOURCE_EXHAUSTED"
	case status == http.StatusServiceUnavailable:
		return "UNAVAILABLE"
	case status >= 500:
		return "INTERNAL"
	default:
		return normalizeErrorType(ir.DialectGemini, typ)
	}
}

func bytesTrimSpace(b []byte) []byte {
	return bytes.TrimSpace(b)
}

func looksLikePlainText(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '{' || s[0] == '[' {
		return false
	}
	return true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
