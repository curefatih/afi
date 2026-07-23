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

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/dataplane/openaichat"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/modelcatalog"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/usage"
)

// ErrStreamUnsupported is returned when the provider capabilities disallow streaming.
var ErrStreamUnsupported = errors.New("streaming is not supported for this provider")

const (
	ModalityChat     = "chat"
	ModalityMessages = "messages"
	ModalityTTS      = "tts"
	ModalitySTT      = "stt"
	ModalityMCP      = "mcp"
)

// UsageEvent is an alias for the canonical usage.Event emitted on the request path.
type UsageEvent = usage.Event

type Pipeline struct {
	Holder      *Holder
	Providers   *Registry
	Hooks       *HookChain
	Wasm        *WasmRunner
	Log         *slog.Logger
	Usage       func(UsageEvent)
	Counters    CounterStore
	Policies    *policy.Evaluator
	Credentials secrets.CredentialOpener
	Secrets     secrets.Resolver
	HTTP        *http.Client
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
	mux.HandleFunc("POST /mcp/{alias}", p.handleMCP)
	mux.HandleFunc("GET /mcp/{alias}", p.handleMCP)
	mux.HandleFunc("DELETE /mcp/{alias}", p.handleMCP)
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		allowHeaders := "Authorization, Content-Type, X-AFI-Tags"
		if reqHdrs := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers")); reqHdrs != "" {
			allowHeaders = reqHdrs
		}
		w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
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
	infos := append([]HookInfo(nil), builtinHookInfos()...)
	if p.Hooks != nil {
		infos = append(infos, p.Hooks.Infos()...)
	}
	out["hooks"] = infos
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

	call := newCallContext(key, reqBody.Model, "/v1/chat/completions", ModalityChat, reqBody.Stream, body, TagsFromRequest(r))
	call.Headers = HeadersForPolicy(r.Header)
	if !p.gateCall(ctx, w, snap, call) {
		return
	}
	body = call.Body

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
	if p.Wasm != nil {
		body, err = p.Wasm.RunBeforeChat(ctx, snap, call.Principal.OrganizationID, body)
		if err != nil {
			log.Error("wasm before_chat", "err", err)
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]string{"message": "chat hook failed: " + err.Error(), "type": "invalid_request_error"},
			})
			return
		}
	}
	call.Body = body

	retryCfg := snap.ResolveRetry(route)
	log.Info("chat.completions",
		"project_id", key.ProjectID,
		"model", reqBody.Model,
		"target_model", route.TargetModel,
		"provider", provider.ID,
		"fallbacks", len(route.Fallbacks),
		"retry_max_attempts", maxTriesFor(retryCfg),
		"stream", reqBody.Stream,
		"snapshot_version", snap.Version,
	)

	var (
		resp             *http.Response
		lastErr          error
		usedProvider     snapshot.Provider
		usedTarget       string
		usedCredentialID string
		promptTokens     int64
		completionTokens int64
		status           = "ok"
	)

	maxTries := maxTriesFor(retryCfg)
targetLoop:
	for i, attempt := range attempts {
		for try := 0; try < maxTries; try++ {
			if try > 0 {
				if sleepErr := sleepBeforeRetry(ctx, retryCfg, try-1); sleepErr != nil {
					lastErr = sleepErr
					discardResponse(resp)
					resp = nil
					break targetLoop
				}
			}

			bound, credID, bindErr := p.bindProviderSecret(ctx, snap, attempt.Provider, key, policy.Request{
				Model:   reqBody.Model,
				Path:    call.Route.Path,
				Stream:  reqBody.Stream,
				Tags:    call.Tags,
				Headers: call.Headers,
			})
			if bindErr != nil {
				lastErr = bindErr
				log.Warn("credential resolve failed", "provider", attempt.Provider.ID, "err", bindErr, "attempt", i)
				// Credential errors are not upstream retries — move to next target.
				if i+1 < len(attempts) {
					continue targetLoop
				}
				break targetLoop
			}
			usedProvider = bound
			usedTarget = attempt.TargetModel
			usedCredentialID = credID
			resp, lastErr = p.callProvider(ctx, bound, attempt.TargetModel, body, reqBody.Stream, call)
			if lastErr != nil {
				log.Warn("upstream attempt failed", "provider", attempt.Provider.ID, "err", lastErr, "attempt", i, "try", try)
				if errors.Is(lastErr, ErrStreamUnsupported) {
					break targetLoop
				}
				if shouldFailoverError(lastErr) {
					if try+1 < maxTries {
						logRetry(log, attempt.Provider.ID, try, maxTries, 0, lastErr)
						continue
					}
					if i+1 < len(attempts) {
						continue targetLoop
					}
				}
				break targetLoop
			}
			if shouldFailoverStatus(resp.StatusCode) {
				if try+1 < maxTries || i+1 < len(attempts) {
					log.Warn("upstream attempt status", "provider", attempt.Provider.ID, "status", resp.StatusCode, "attempt", i, "try", try)
					code := resp.StatusCode
					discardResponse(resp)
					resp = nil
					if try+1 < maxTries {
						logRetry(log, attempt.Provider.ID, try, maxTries, code, nil)
						continue
					}
					continue targetLoop
				}
			}
			break targetLoop
		}
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
			CredentialID:   usedCredentialID,
			Model:          reqBody.Model,
			ProviderType:   usedProvider.Type,
			TargetModel:    usedTarget,
			Status:         "error",
			LatencyMs:      time.Since(start).Milliseconds(),
			Modality:       ModalityChat,
			Tags:           cloneTags(call.Tags),
		})
		afterInfo := AfterCallInfo{
			Status: "error", LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: usedProvider.Type, TargetModel: usedTarget,
		}
		p.runAfterCall(ctx, snap, call, afterInfo)
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
		applyResponseHeaders(w, call)
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
	} else if reqBody.Stream && resp.StatusCode < 400 {
		applyResponseHeaders(w, call)
		for k, vals := range resp.Header {
			if strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") {
				continue
			}
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		var copyErr error
		promptTokens, completionTokens, copyErr = openaichat.CopySSEAndParseUsage(w, resp.Body)
		if copyErr != nil {
			log.Error("copy stream response", "err", copyErr)
			status = "error"
		}
		if promptTokens+completionTokens > 0 {
			p.incrTokens(ctx, snap, key, promptTokens+completionTokens)
		}
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
		CredentialID:     usedCredentialID,
		Model:            reqBody.Model,
		ProviderType:     usedProvider.Type,
		TargetModel:      usedTarget,
		Status:           status,
		LatencyMs:        time.Since(start).Milliseconds(),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Modality:         ModalityChat,
		Metrics:          tokenMetrics(promptTokens, completionTokens),
		Tags:             cloneTags(call.Tags),
	})
	afterInfo := AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: usedProvider.Type, TargetModel: usedTarget,
		PromptTokens: promptTokens, CompletionTokens: completionTokens,
	}
	p.runAfterCall(ctx, snap, call, afterInfo)
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
		providerType := "openai"
		caps := snapshot.DefaultCapabilities(providerType)
		if prov, ok := snap.Providers[route.ProviderID]; ok {
			providerType = prov.Type
			caps = snapshot.NormalizeCapabilities(prov.Type, prov.Capabilities)
		}
		item := modelListItem(route.Model, route.TargetModel, providerType, caps)
		data = append(data, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   data,
	})
}

// modelListItem builds a /v1/models entry from route + provider caps + curated catalog.
func modelListItem(virtualModel, targetModel, providerType string, caps snapshot.ProviderCapabilities) map[string]any {
	mode := modelcatalog.ModeChat
	chat := caps.Chat
	stream := caps.Stream && caps.Chat
	tts := caps.TTS && modelLooksLikeTTS(virtualModel, targetModel)
	stt := caps.STT && modelLooksLikeSTT(virtualModel, targetModel)
	var maxIn, maxOut int
	var supportsVision, supportsTools bool

	if entry, ok := modelcatalog.Lookup(providerType, targetModel); ok {
		mode = entry.Mode
		if mode == "" {
			mode = modelcatalog.ModeChat
		}
		maxIn = entry.MaxInputTokens
		maxOut = entry.MaxOutputTokens
		supportsVision = entry.SupportsVision
		supportsTools = entry.SupportsTools
		switch {
		case entry.IsTTS():
			chat, stream, tts, stt = false, false, caps.TTS, false
		case entry.IsSTT():
			chat, stream, tts, stt = false, false, false, caps.STT
		default:
			chat = caps.Chat
			stream = caps.Stream && caps.Chat && entry.StreamingEnabled()
			tts, stt = false, false
		}
	} else {
		switch {
		case tts:
			mode = modelcatalog.ModeAudioSpeech
			chat, stream, stt = false, false, false
		case stt:
			mode = modelcatalog.ModeAudioTranscription
			chat, stream, tts = false, false, false
		default:
			mode = modelcatalog.ModeChat
			tts, stt = false, false
		}
	}

	item := map[string]any{
		"id":                 virtualModel,
		"object":             "model",
		"owned_by":           "afi",
		"mode":               mode,
		"supports_streaming": stream,
		"supports_tts":       tts,
		"supports_stt":       stt,
		"capabilities": map[string]bool{
			"chat":   chat,
			"stream": stream,
			"tts":    tts,
			"stt":    stt,
		},
	}
	if maxIn > 0 {
		item["max_input_tokens"] = maxIn
	}
	if maxOut > 0 {
		item["max_output_tokens"] = maxOut
	}
	if supportsVision {
		item["supports_vision"] = true
	}
	if supportsTools {
		item["supports_tools"] = true
	}
	return item
}

func (p *Pipeline) callProvider(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool, call *CallContext) (*http.Response, error) {
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
	if call != nil {
		ctx = llm.WithExtraHeaders(ctx, call.RequestHeaders)
	}
	return adapter.Chat(ctx, provider, targetModel, body, stream)
}

// bindProviderSecret resolves BYOK via a matching use_credential policy action,
// else api_key → project → org assignment, else provider.api_key_env.
func (p *Pipeline) bindProviderSecret(ctx context.Context, snap *snapshot.Snapshot, provider snapshot.Provider, key snapshot.APIKey, req policy.Request) (snapshot.Provider, string, error) {
	if snap != nil {
		overrideName := ""
		if p.Policies != nil && len(snap.Policies) > 0 {
			d, err := p.Policies.Apply(snap.Policies, key, req, policy.Credential{})
			if err != nil {
				return provider, "", err
			}
			if !d.Allowed {
				msg := "request denied by policy"
				if d.DeniedBy != "" {
					msg = "request denied by policy: " + d.DeniedBy
				}
				return provider, "", fmt.Errorf("%s", msg)
			}
			overrideName = d.CredentialName
		}
		credMeta := policy.CredentialFromSnapshot(snap, provider.Type, key, overrideName)
		if p.Policies != nil && len(snap.Policies) > 0 {
			d, err := p.Policies.Apply(snap.Policies, key, req, credMeta)
			if err != nil {
				return provider, "", err
			}
			if !d.Allowed {
				msg := "request denied by policy"
				if d.DeniedBy != "" {
					msg = "request denied by policy: " + d.DeniedBy
				}
				return provider, "", fmt.Errorf("%s", msg)
			}
			if d.CredentialName != "" {
				overrideName = d.CredentialName
			}
		}
		cred, ok, missingOverride := snap.ResolveCredentialForCall(provider.Type, key, overrideName)
		if missingOverride {
			return provider, "", fmt.Errorf("no credential named %q for provider type %q", overrideName, provider.Type)
		}
		if ok {
			if p.Credentials == nil {
				return provider, "", fmt.Errorf("credential resolver not configured")
			}
			secret, err := p.Credentials.Open(ctx, cred)
			if err != nil {
				return provider, "", err
			}
			provider.InlineAPIKey = secret
			return provider, cred.ID, nil
		}
	}
	if strings.TrimSpace(provider.APIKeyEnv) == "" {
		return provider, "", fmt.Errorf("no credential assigned for provider type %q and no api_key_env fallback", provider.Type)
	}
	return provider, "", nil
}

func (p *Pipeline) recordUsage(e UsageEvent) {
	e.MarkBYOK()
	if p.Usage != nil {
		p.Usage(e)
	}
	p.Log.Info("usage",
		"organization_id", e.OrganizationID,
		"project_id", e.ProjectID,
		"api_key_id", e.APIKeyID,
		"credential_id", e.CredentialID,
		"used_byok", e.UsedBYOK,
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
