package dataplane

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/kernel"
)

const mcpBodyLimit = 8 << 20 // 8 MiB

func (p *Pipeline) handleMCP(w http.ResponseWriter, r *http.Request) {
	reqID := kernel.NewRequestID()
	ctx := kernel.WithRequestID(r.Context(), reqID)
	log := p.Log.With("request_id", reqID)
	start := time.Now()

	alias := strings.TrimSpace(r.PathValue("alias"))
	if alias == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "alias required", "type": "invalid_request_error"},
		})
		return
	}

	rawKey, err := bearerToken(r.Header.Get("Authorization"))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "missing or invalid authorization", "type": "invalid_request_error"},
		})
		return
	}
	snap := p.Holder.Get()
	if snap == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{"message": "no snapshot loaded", "type": "server_error"},
		})
		return
	}
	key, ok := snap.LookupKey(rawKey)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": map[string]string{"message": "invalid api key", "type": "invalid_request_error"},
		})
		return
	}

	var body []byte
	if r.Method == http.MethodPost {
		body, err = io.ReadAll(io.LimitReader(r.Body, mcpBodyLimit))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]string{"message": "failed to read body", "type": "invalid_request_error"},
			})
			return
		}
	}

	path := "/mcp/" + alias
	stream := strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream")
	call := newCallContext(key, alias, path, ModalityMCP, stream, body, TagsFromRequest(r))
	call.Headers = HeadersForPolicy(r.Header)
	if !p.gateCall(ctx, w, snap, call) {
		return
	}
	body = call.Body

	backend, ok := snap.LookupMCPBackend(key.OrganizationID, alias)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]string{"message": "unknown mcp alias", "type": "invalid_request_error"},
		})
		return
	}

	method, tool := parseMCPJSONRPCMeta(body)
	if !mcpMethodAllowed(backend.MethodAllowlist, method) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": map[string]string{"message": "mcp method not allowed: " + method, "type": "invalid_request_error"},
		})
		return
	}

	if backend.APIKeyEnv != "" {
		secret, err := p.resolveMCPSecret(ctx, backend.APIKeyEnv)
		if err != nil {
			log.Error("mcp secret", "err", err)
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": map[string]string{"message": "upstream credential unavailable", "type": "server_error"},
			})
			return
		}
		backend.InlineAPIKey = secret
	}

	upReq, err := http.NewRequestWithContext(ctx, r.Method, backend.BaseURL, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "failed to build upstream request", "type": "server_error"},
		})
		return
	}
	if q := r.URL.RawQuery; q != "" {
		upReq.URL.RawQuery = q
	}
	copyMCPRequestHeaders(upReq.Header, r.Header, call.RequestHeaders)
	if backend.InlineAPIKey != "" {
		upReq.Header.Set("Authorization", "Bearer "+backend.InlineAPIKey)
	}
	if len(body) > 0 && upReq.Header.Get("Content-Type") == "" {
		upReq.Header.Set("Content-Type", "application/json")
	}

	client := p.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(upReq)
	status := "ok"
	metrics := map[string]any{}
	if method != "" {
		metrics["method"] = method
	}
	if tool != "" {
		metrics["tool"] = tool
	}
	if err != nil {
		log.Error("mcp upstream", "err", err, "alias", alias)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		status = "error"
		metrics["latency_ms"] = time.Since(start).Milliseconds()
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID,
			Model: alias, ProviderType: "mcp", TargetModel: backend.ID,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalityMCP, Metrics: metrics, Tags: cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: "mcp", TargetModel: backend.ID,
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		status = "error"
	}
	applyResponseHeaders(w, call)
	if err := CopyResponse(w, resp); err != nil {
		log.Error("copy mcp response", "err", err)
		status = "error"
	}
	metrics["latency_ms"] = time.Since(start).Milliseconds()
	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID,
		Model: alias, ProviderType: "mcp", TargetModel: backend.ID,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalityMCP, Metrics: metrics, Tags: cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: "mcp", TargetModel: backend.ID,
	})
}

func (p *Pipeline) resolveMCPSecret(ctx context.Context, envName string) (string, error) {
	if p.Secrets != nil {
		return p.Secrets.Get(ctx, envName)
	}
	return secrets.Env{}.Get(ctx, envName)
}

func copyMCPRequestHeaders(dst, src http.Header, extra map[string]string) {
	for _, k := range []string{
		"Accept", "Content-Type", "Mcp-Session-Id", "Last-Event-ID",
	} {
		if v := src.Get(k); v != "" {
			dst.Set(k, v)
		}
	}
	for k, v := range extra {
		if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Origin") {
			continue
		}
		if v != "" {
			dst.Set(k, v)
		}
	}
}

func parseMCPJSONRPCMeta(body []byte) (method, tool string) {
	if len(body) == 0 {
		return "", ""
	}
	var msg struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", ""
	}
	method = strings.TrimSpace(msg.Method)
	if method == "tools/call" && len(msg.Params) > 0 {
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			tool = strings.TrimSpace(params.Name)
		}
	}
	return method, tool
}

func mcpMethodAllowed(allowlist []string, method string) bool {
	if len(allowlist) == 0 || method == "" {
		return true
	}
	for _, m := range allowlist {
		if m == method {
			return true
		}
		// Prefix wildcards like "resources/*"
		if strings.HasSuffix(m, "/*") {
			prefix := strings.TrimSuffix(m, "*")
			if strings.HasPrefix(method, prefix) {
				return true
			}
		}
	}
	return false
}
