package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
)

const protocolProbeTimeout = 8 * time.Second
const protocolProbeBodyLimit = 64 << 10 // 64 KiB

// ProtocolProbeResult is returned by MCP/A2A connection tests.
type ProtocolProbeResult struct {
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code,omitempty"`
	LatencyMs  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

func resolveProbeSecret(ctx context.Context, apiKeyEnv string) (string, error) {
	apiKeyEnv = strings.TrimSpace(apiKeyEnv)
	if apiKeyEnv == "" {
		return "", nil
	}
	return secrets.Env{}.Get(ctx, apiKeyEnv)
}

func probeHTTP(ctx context.Context, method, url string, body []byte, headers map[string]string) ProtocolProbeResult {
	start := time.Now()
	reqCtx, cancel := context.WithTimeout(ctx, protocolProbeTimeout)
	defer cancel()

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return ProtocolProbeResult{OK: false, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error()}
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProtocolProbeResult{OK: false, LatencyMs: latency, Error: err.Error()}
	}
	defer resp.Body.Close()
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, protocolProbeBodyLimit))

	// Reachable if we got an HTTP response (including 4xx/5xx auth/protocol errors).
	ok := resp.StatusCode > 0
	detail := strings.TrimSpace(string(snippet))
	if len(detail) > 200 {
		detail = detail[:200] + "…"
	}
	out := ProtocolProbeResult{
		OK:         ok,
		StatusCode: resp.StatusCode,
		LatencyMs:  latency,
		Detail:     detail,
	}
	if resp.StatusCode >= 500 {
		out.OK = false
		out.Error = fmt.Sprintf("upstream returned %d", resp.StatusCode)
	}
	return out
}

func probeMCP(ctx context.Context, baseURL, apiKeyEnv string) (ProtocolProbeResult, error) {
	base, err := gatewayconfig.ParseMCPBaseURL(baseURL)
	if err != nil {
		return ProtocolProbeResult{}, err
	}
	secret, err := resolveProbeSecret(ctx, apiKeyEnv)
	if err != nil {
		return ProtocolProbeResult{OK: false, Error: "credential unavailable: " + err.Error()}, nil
	}
	initBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "afi", "version": "probe"},
		},
	})
	headers := map[string]string{
		"Accept":       "application/json, text/event-stream",
		"Content-Type": "application/json",
	}
	if secret != "" {
		headers["Authorization"] = "Bearer " + secret
	}
	return probeHTTP(ctx, http.MethodPost, base, initBody, headers), nil
}

func probeA2A(ctx context.Context, upstreamURL, cardURL, apiKeyEnv string) (ProtocolProbeResult, error) {
	upstream, err := gatewayconfig.ParseA2AURL(upstreamURL)
	if err != nil {
		return ProtocolProbeResult{}, err
	}
	cardURL = strings.TrimSpace(cardURL)
	if cardURL == "" {
		cardURL = strings.TrimRight(upstream, "/") + "/.well-known/agent-card.json"
	} else {
		cardURL, err = gatewayconfig.ParseA2AURL(cardURL)
		if err != nil {
			return ProtocolProbeResult{}, fmt.Errorf("%w: card_url invalid", kernel.ErrInvalidRequest)
		}
	}
	secret, err := resolveProbeSecret(ctx, apiKeyEnv)
	if err != nil {
		return ProtocolProbeResult{OK: false, Error: "credential unavailable: " + err.Error()}, nil
	}
	headers := map[string]string{"Accept": "application/json"}
	if secret != "" {
		headers["Authorization"] = "Bearer " + secret
	}
	return probeHTTP(ctx, http.MethodGet, cardURL, nil, headers), nil
}
