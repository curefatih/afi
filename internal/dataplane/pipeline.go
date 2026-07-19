package dataplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type UsageEvent struct {
	OrganizationID   string
	ProjectID        string
	APIKeyID         string
	Model            string
	ProviderType     string
	TargetModel      string
	Status           string
	LatencyMs        int64
	PromptTokens     int64
	CompletionTokens int64
}

type Pipeline struct {
	Holder    *Holder
	OpenAI    *OpenAIClient
	Anthropic *AnthropicClient
	Log       *slog.Logger
	Usage     func(UsageEvent)
	Counters  CounterStore
}

func NewPipeline(holder *Holder, openai *OpenAIClient, log *slog.Logger) *Pipeline {
	return &Pipeline{
		Holder:    holder,
		OpenAI:    openai,
		Anthropic: NewAnthropicClient(),
		Log:       log,
	}
}

func (p *Pipeline) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", p.handleHealth)
	mux.HandleFunc("POST /v1/chat/completions", p.handleChatCompletions)
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p *Pipeline) handleHealth(w http.ResponseWriter, r *http.Request) {
	snap := p.Holder.Get()
	out := map[string]any{"status": "ok"}
	if snap != nil {
		out["snapshot_version"] = snap.Version
	} else {
		out["snapshot_version"] = nil
	}
	writeJSON(w, http.StatusOK, out)
}

type routeAttempt struct {
	Provider    snapshot.Provider
	TargetModel string
}

func (p *Pipeline) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	reqID := kernel.NewRequestID()
	ctx := kernel.WithRequestID(r.Context(), reqID)
	log := p.Log.With("request_id", reqID)
	start := time.Now()

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

	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "failed to read body", "type": "invalid_request_error"},
		})
		return
	}

	var reqBody struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(body, &reqBody); err != nil || reqBody.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "model is required", "type": "invalid_request_error"},
		})
		return
	}

	denied, err := p.checkAndIncrRequests(ctx, snap, key)
	if err != nil {
		log.Error("quota check", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "quota check failed", "type": "server_error"},
		})
		return
	}
	if denied {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error": map[string]string{
				"message": "quota exceeded",
				"type":    "insufficient_quota",
				"code":    "insufficient_quota",
			},
		})
		return
	}

	route, provider, ok := snap.LookupRoute(key.OrganizationID, reqBody.Model)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no route for model", "type": "invalid_request_error"},
		})
		return
	}

	attempts := buildAttempts(snap, route, provider)
	if len(attempts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no usable providers for route", "type": "invalid_request_error"},
		})
		return
	}

	log.Info("chat.completions",
		"project_id", key.ProjectID,
		"model", reqBody.Model,
		"target_model", route.TargetModel,
		"provider", provider.ID,
		"fallbacks", len(route.Fallbacks),
		"stream", reqBody.Stream,
		"snapshot_version", snap.Version,
	)

	var (
		resp             *http.Response
		lastErr          error
		usedProvider     snapshot.Provider
		usedTarget       string
		promptTokens     int64
		completionTokens int64
		status           = "ok"
	)

	for i, attempt := range attempts {
		usedProvider = attempt.Provider
		usedTarget = attempt.TargetModel
		resp, lastErr = p.callProvider(ctx, attempt.Provider, attempt.TargetModel, body, reqBody.Stream)
		if lastErr != nil {
			log.Warn("upstream attempt failed", "provider", attempt.Provider.ID, "err", lastErr, "attempt", i)
			if i+1 < len(attempts) && shouldFailoverError(lastErr) {
				continue
			}
			break
		}
		if shouldFailoverStatus(resp.StatusCode) && i+1 < len(attempts) {
			log.Warn("upstream attempt status", "provider", attempt.Provider.ID, "status", resp.StatusCode, "attempt", i)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			resp = nil
			continue
		}
		break
	}

	if lastErr != nil && resp == nil {
		log.Error("upstream error", "err", lastErr)
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID,
			ProjectID:      key.ProjectID,
			APIKeyID:       key.ID,
			Model:          reqBody.Model,
			ProviderType:   usedProvider.Type,
			TargetModel:    usedTarget,
			Status:         "error",
			LatencyMs:      time.Since(start).Milliseconds(),
		})
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": lastErr.Error(), "type": "server_error"},
		})
		return
	}
	if resp == nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": "all upstream attempts failed", "type": "server_error"},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		status = "upstream_error"
	}

	if !reqBody.Stream && resp.StatusCode < 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": map[string]string{"message": "failed to read upstream", "type": "server_error"},
			})
			return
		}
		promptTokens, completionTokens = parseUsageTokens(respBody)
		p.incrTokens(ctx, snap, key, promptTokens+completionTokens)
		for k, vals := range resp.Header {
			if strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") {
				continue
			}
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBody)
	} else {
		if err := CopyResponse(w, resp); err != nil {
			log.Error("copy response", "err", err)
			status = "error"
		}
	}

	p.recordUsage(UsageEvent{
		OrganizationID:   key.OrganizationID,
		ProjectID:        key.ProjectID,
		APIKeyID:         key.ID,
		Model:            reqBody.Model,
		ProviderType:     usedProvider.Type,
		TargetModel:      usedTarget,
		Status:           status,
		LatencyMs:        time.Since(start).Milliseconds(),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	})
}

func buildAttempts(snap *snapshot.Snapshot, route snapshot.Route, primary snapshot.Provider) []routeAttempt {
	out := []routeAttempt{{Provider: primary, TargetModel: route.TargetModel}}
	for _, fb := range route.Fallbacks {
		p, ok := snap.Providers[fb.ProviderID]
		if !ok {
			continue
		}
		target := fb.TargetModel
		if target == "" {
			target = route.TargetModel
		}
		out = append(out, routeAttempt{Provider: p, TargetModel: target})
	}
	return out
}

func shouldFailoverStatus(code int) bool {
	return code >= 500 || code == http.StatusTooManyRequests
}

func shouldFailoverError(err error) bool {
	return err != nil
}

func (p *Pipeline) callProvider(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	switch provider.Type {
	case "openai":
		if p.OpenAI == nil {
			return nil, fmt.Errorf("openai client not configured")
		}
		return p.OpenAI.ChatCompletions(ctx, provider, targetModel, body, stream)
	case "anthropic":
		if stream {
			return nil, fmt.Errorf("anthropic streaming is not supported")
		}
		if p.Anthropic == nil {
			return nil, fmt.Errorf("anthropic client not configured")
		}
		return p.Anthropic.Messages(ctx, provider, targetModel, body)
	default:
		return nil, fmt.Errorf("unsupported provider type %q", provider.Type)
	}
}

func (p *Pipeline) recordUsage(e UsageEvent) {
	if p.Usage != nil {
		p.Usage(e)
	}
	p.Log.Info("usage",
		"organization_id", e.OrganizationID,
		"project_id", e.ProjectID,
		"api_key_id", e.APIKeyID,
		"model", e.Model,
		"provider_type", e.ProviderType,
		"target_model", e.TargetModel,
		"status", e.Status,
		"latency_ms", e.LatencyMs,
		"prompt_tokens", e.PromptTokens,
		"completion_tokens", e.CompletionTokens,
	)
}

func parseUsageTokens(body []byte) (prompt, completion int64) {
	var parsed struct {
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0
	}
	return parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens
}

func bearerToken(h string) (string, error) {
	if !strings.HasPrefix(h, "Bearer ") {
		return "", errors.New("missing bearer")
	}
	tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	if tok == "" {
		return "", errors.New("empty token")
	}
	return tok, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// AuthenticateKey is exported for unit tests.
func AuthenticateKey(snap *snapshot.Snapshot, rawKey string) (snapshot.APIKey, error) {
	if snap == nil {
		return snapshot.APIKey{}, kernel.ErrNotFound
	}
	k, ok := snap.LookupKey(rawKey)
	if !ok {
		return snapshot.APIKey{}, kernel.ErrUnauthorized
	}
	return k, nil
}
