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

	"github.com/curefatih/afi/internal/dataplane/openaichat"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

// ErrStreamUnsupported is returned when the provider capabilities disallow streaming.
var ErrStreamUnsupported = errors.New("streaming is not supported for this provider")

const (
	ModalityChat     = "chat"
	ModalityMessages = "messages"
	ModalityTTS      = "tts"
	ModalitySTT      = "stt"
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
	Modality         string
	Metrics          map[string]any
}

type Pipeline struct {
	Holder    *Holder
	Providers *Registry
	Hooks     *HookChain
	Log       *slog.Logger
	Usage     func(UsageEvent)
	Counters  CounterStore
	Policies  *policy.Evaluator
}

// NewPipeline builds a pipeline with an explicit provider registry.
// Built-in LLM adapters are registered from cmd/gateway via adapters/llm.
func NewPipeline(holder *Holder, reg *Registry, log *slog.Logger) *Pipeline {
	if reg == nil {
		reg = NewRegistry()
	}
	return &Pipeline{Holder: holder, Providers: reg, Log: log}
}

// NewPipelineWithRegistry uses an explicit provider registry.
func NewPipelineWithRegistry(holder *Holder, reg *Registry, log *slog.Logger) *Pipeline {
	return NewPipeline(holder, reg, log)
}

func (p *Pipeline) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", p.handleHealth)
	mux.HandleFunc("GET /v1/models", p.handleModels)
	mux.HandleFunc("POST /v1/chat/completions", p.handleChatCompletions)
	mux.HandleFunc("POST /v1/messages", p.handleMessages)
	mux.HandleFunc("POST /v1/audio/speech", p.handleAudioSpeech)
	mux.HandleFunc("POST /v1/audio/transcriptions", p.handleAudioTranscriptions)
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
	if p.Providers != nil {
		out["provider_types"] = p.Providers.Types()
	}
	if infos := p.Hooks.Infos(); len(infos) > 0 {
		out["hooks"] = infos
	} else {
		out["hooks"] = []HookInfo{}
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

	if !p.checkPolicies(w, snap, key, reqBody.Model, "/v1/chat/completions", reqBody.Stream) {
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

	body, err = p.Hooks.RunBeforeChat(ctx, body)
	if err != nil {
		log.Error("chat hook", "err", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "chat hook failed: " + err.Error(), "type": "invalid_request_error"},
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
			if errors.Is(lastErr, ErrStreamUnsupported) {
				break
			}
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
		if errors.Is(lastErr, ErrStreamUnsupported) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]string{"message": lastErr.Error(), "type": "invalid_request_error"},
			})
			return
		}
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
			Modality:       ModalityChat,
		})
		p.Hooks.RunAfterChat(ctx, AfterChatInfo{
			Model: reqBody.Model, Status: "error",
			LatencyMs:    time.Since(start).Milliseconds(),
			ProviderType: usedProvider.Type, TargetModel: usedTarget,
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
		promptTokens, completionTokens = openaichat.ParseUsageTokens(respBody)
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
		Modality:         ModalityChat,
		Metrics:          tokenMetrics(promptTokens, completionTokens),
	})
	p.Hooks.RunAfterChat(ctx, AfterChatInfo{
		Model: reqBody.Model, Status: status,
		LatencyMs:    time.Since(start).Milliseconds(),
		ProviderType: usedProvider.Type, TargetModel: usedTarget,
	})
}

func tokenMetrics(prompt, completion int64) map[string]any {
	if prompt == 0 && completion == 0 {
		return nil
	}
	return map[string]any{
		"prompt_tokens":     prompt,
		"completion_tokens": completion,
	}
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
	return err != nil && !errors.Is(err, ErrStreamUnsupported)
}

func (p *Pipeline) handleModels(w http.ResponseWriter, r *http.Request) {
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

	data := make([]map[string]any, 0)
	for _, route := range snap.Routes {
		if route.OrganizationID != key.OrganizationID {
			continue
		}
		caps := snapshot.DefaultCapabilities("openai")
		if prov, ok := snap.Providers[route.ProviderID]; ok {
			caps = snapshot.NormalizeCapabilities(prov.Type, prov.Capabilities)
		}
		tts := caps.TTS && modelLooksLikeTTS(route.Model, route.TargetModel)
		stt := caps.STT && modelLooksLikeSTT(route.Model, route.TargetModel)
		data = append(data, map[string]any{
			"id":                 route.Model,
			"object":             "model",
			"owned_by":           "afi",
			"supports_streaming": caps.Stream && caps.Chat,
			"supports_tts":       tts,
			"supports_stt":       stt,
			"capabilities": map[string]bool{
				"chat":   caps.Chat,
				"stream": caps.Stream,
				"tts":    tts,
				"stt":    stt,
			},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   data,
	})
}

func (p *Pipeline) callProvider(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	if p.Providers == nil {
		return nil, fmt.Errorf("provider registry not configured")
	}
	adapter, ok := p.Providers.Get(provider.Type)
	if !ok {
		return nil, fmt.Errorf("unsupported provider type %q", provider.Type)
	}
	caps := snapshot.NormalizeCapabilities(provider.Type, provider.Capabilities)
	if stream && !caps.Stream {
		return nil, fmt.Errorf("%w", ErrStreamUnsupported)
	}
	if !caps.Chat {
		return nil, fmt.Errorf("chat is not supported for provider type %q", provider.Type)
	}
	return adapter.Chat(ctx, provider, targetModel, body, stream)
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
