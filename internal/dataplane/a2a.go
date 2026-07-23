package dataplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

const a2aBodyLimit = 8 << 20 // 8 MiB

func (p *Pipeline) handleA2AJSONRPC(w http.ResponseWriter, r *http.Request) {
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

	body, err := io.ReadAll(io.LimitReader(r.Body, a2aBodyLimit))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "failed to read body", "type": "invalid_request_error"},
		})
		return
	}

	path := "/a2a/" + alias
	stream := strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream")
	call := newCallContext(key, alias, path, ModalityA2A, stream, body, TagsFromRequest(r))
	call.Headers = HeadersForPolicy(r.Header)
	if !p.gateCall(ctx, w, snap, call) {
		return
	}
	body = call.Body

	agent, ok := snap.LookupA2AAgent(key.OrganizationID, alias)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]string{"message": "unknown a2a alias", "type": "invalid_request_error"},
		})
		return
	}

	method, skill, taskID := parseA2AJSONRPCMeta(body)
	if err := p.bindA2ASecret(ctx, &agent); err != nil {
		log.Error("a2a secret", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": "upstream credential unavailable", "type": "server_error"},
		})
		return
	}

	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, agent.UpstreamURL, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "failed to build upstream request", "type": "server_error"},
		})
		return
	}
	if q := r.URL.RawQuery; q != "" {
		upReq.URL.RawQuery = q
	}
	copyA2ARequestHeaders(upReq.Header, r.Header, call.RequestHeaders)
	if agent.InlineAPIKey != "" {
		upReq.Header.Set("Authorization", "Bearer "+agent.InlineAPIKey)
	}
	if upReq.Header.Get("Content-Type") == "" {
		upReq.Header.Set("Content-Type", "application/json")
	}

	p.proxyA2AResponse(ctx, w, snap, call, key, agent, alias, method, skill, taskID, upReq, start, log)
}

func (p *Pipeline) handleA2AAgentCard(w http.ResponseWriter, r *http.Request) {
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

	path := "/a2a/" + alias + "/.well-known/agent-card.json"
	call := newCallContext(key, alias, path, ModalityA2A, false, nil, TagsFromRequest(r))
	call.Headers = HeadersForPolicy(r.Header)
	if !p.gateCall(ctx, w, snap, call) {
		return
	}

	agent, ok := snap.LookupA2AAgent(key.OrganizationID, alias)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]string{"message": "unknown a2a alias", "type": "invalid_request_error"},
		})
		return
	}

	if err := p.bindA2ASecret(ctx, &agent); err != nil {
		log.Error("a2a secret", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": "upstream credential unavailable", "type": "server_error"},
		})
		return
	}

	card, err := p.loadA2AAgentCard(ctx, agent)
	status := "ok"
	if err != nil {
		log.Error("a2a agent card", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": "failed to load agent card", "type": "server_error"},
		})
		status = "error"
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
			Model: alias, ProviderType: "a2a", TargetModel: agent.ID,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalityA2A, Metrics: map[string]any{"method": "agent-card"}, Tags: cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: "a2a", TargetModel: agent.ID,
		})
		return
	}

	gatewayURL := publicGatewayBase(r) + "/a2a/" + alias
	rewritten, err := rewriteA2AAgentCard(card, gatewayURL)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": "invalid agent card", "type": "server_error"},
		})
		return
	}

	applyResponseHeaders(w, call)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(rewritten)

	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
		Model: alias, ProviderType: "a2a", TargetModel: agent.ID,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalityA2A, Metrics: map[string]any{"method": "agent-card"}, Tags: cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: "a2a", TargetModel: agent.ID,
	})
}

func (p *Pipeline) bindA2ASecret(ctx context.Context, agent *snapshot.A2AAgent) error {
	if agent.APIKeyEnv == "" {
		return nil
	}
	secret, err := p.resolveMCPSecret(ctx, agent.APIKeyEnv)
	if err != nil {
		return err
	}
	agent.InlineAPIKey = secret
	return nil
}

func (p *Pipeline) loadA2AAgentCard(ctx context.Context, agent snapshot.A2AAgent) ([]byte, error) {
	if len(agent.CardCache) > 0 {
		return agent.CardCache, nil
	}
	cardURL := strings.TrimSpace(agent.CardURL)
	if cardURL == "" {
		cardURL = strings.TrimRight(agent.UpstreamURL, "/") + "/.well-known/agent-card.json"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cardURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if agent.InlineAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+agent.InlineAPIKey)
	}
	client := p.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, a2aBodyLimit))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream card status %d", resp.StatusCode)
	}
	return body, nil
}

func (p *Pipeline) proxyA2AResponse(
	ctx context.Context,
	w http.ResponseWriter,
	snap *snapshot.Snapshot,
	call *CallContext,
	key snapshot.APIKey,
	agent snapshot.A2AAgent,
	alias, method, skill, taskID string,
	upReq *http.Request,
	start time.Time,
	log interface{ Error(string, ...any) },
) {
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
	if skill != "" {
		metrics["skill"] = skill
	}
	if taskID != "" {
		metrics["task_id"] = taskID
	}
	if err != nil {
		log.Error("a2a upstream", "err", err, "alias", alias)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		status = "error"
		metrics["latency_ms"] = time.Since(start).Milliseconds()
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
			Model: alias, ProviderType: "a2a", TargetModel: agent.ID,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalityA2A, Metrics: metrics, Tags: cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: "a2a", TargetModel: agent.ID,
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		status = "error"
	}
	applyResponseHeaders(w, call)
	if err := CopyResponse(w, resp); err != nil {
		log.Error("copy a2a response", "err", err)
		status = "error"
	}
	metrics["latency_ms"] = time.Since(start).Milliseconds()
	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
		Model: alias, ProviderType: "a2a", TargetModel: agent.ID,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalityA2A, Metrics: metrics, Tags: cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: "a2a", TargetModel: agent.ID,
	})
}

func copyA2ARequestHeaders(dst, src http.Header, extra map[string]string) {
	for _, k := range []string{"Accept", "Content-Type"} {
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

func parseA2AJSONRPCMeta(body []byte) (method, skill, taskID string) {
	if len(body) == 0 {
		return "", "", ""
	}
	var msg struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", "", ""
	}
	method = strings.TrimSpace(msg.Method)
	if len(msg.Params) == 0 {
		return method, "", ""
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return method, "", ""
	}
	if raw, ok := params["id"]; ok {
		var id string
		if json.Unmarshal(raw, &id) == nil {
			taskID = strings.TrimSpace(id)
		}
	}
	if raw, ok := params["message"]; ok {
		var message struct {
			Metadata map[string]any `json:"metadata"`
		}
		if json.Unmarshal(raw, &message) == nil && message.Metadata != nil {
			if s, ok := message.Metadata["skill"].(string); ok {
				skill = strings.TrimSpace(s)
			}
		}
	}
	if raw, ok := params["skill"]; ok {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			skill = strings.TrimSpace(s)
		}
	}
	return method, skill, taskID
}

func publicGatewayBase(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = strings.Split(proto, ",")[0]
		scheme = strings.TrimSpace(scheme)
	}
	host := r.Host
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); fwd != "" {
		host = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	return scheme + "://" + host
}

func rewriteA2AAgentCard(card []byte, gatewayURL string) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal(card, &obj); err != nil {
		return nil, err
	}
	obj["url"] = gatewayURL
	// Prefer common alternate endpoint fields when present.
	if _, ok := obj["endpoint"]; ok {
		obj["endpoint"] = gatewayURL
	}
	if interfaces, ok := obj["interfaces"].([]any); ok {
		for _, item := range interfaces {
			if m, ok := item.(map[string]any); ok {
				if _, has := m["url"]; has {
					m["url"] = gatewayURL
				}
			}
		}
	}
	return json.Marshal(obj)
}
