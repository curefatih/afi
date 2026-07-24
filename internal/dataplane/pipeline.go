package dataplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/dataplane/routing"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/modelcatalog"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/telemetry"
	"github.com/curefatih/afi/internal/usage"
)

// ErrStreamUnsupported is returned when the provider capabilities disallow streaming.
var ErrStreamUnsupported = errors.New("streaming is not supported for this provider")

const (
	ModalityChat      = "chat"
	ModalityMessages  = "messages"
	ModalityTTS       = "tts"
	ModalitySTT       = "stt"
	ModalityEmbedding = "embedding"
	ModalityImage     = "image"
	ModalityMCP       = "mcp"
	ModalityA2A       = "a2a"
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
	Metrics     *telemetry.GatewayMetrics
	// RouteRand optional RNG for weighted routing (tests); nil uses math/rand global.
	RouteRand *rand.Rand
	// RouteSignals optional gateway-local EWMA store for latency/cost adaptive routing.
	RouteSignals routing.SignalStore
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
	mux.HandleFunc("GET /openai/v1/models", p.handleModels)
	mux.HandleFunc("POST /v1/chat/completions", p.handleChatCompletions)
	mux.HandleFunc("POST /openai/v1/chat/completions", p.handleChatCompletions)
	mux.HandleFunc("POST /v1/messages", p.handleMessages)
	mux.HandleFunc("POST /anthropic/v1/messages", p.handleMessages)
	mux.HandleFunc("POST /v1/embeddings", p.handleEmbeddings)
	mux.HandleFunc("POST /v1/images/generations", p.handleImagesGenerations)
	mux.HandleFunc("POST /v1/audio/speech", p.handleAudioSpeech)
	mux.HandleFunc("POST /v1/audio/transcriptions", p.handleAudioTranscriptions)
	mux.HandleFunc("POST /mcp/{alias}", p.handleMCP)
	mux.HandleFunc("GET /mcp/{alias}", p.handleMCP)
	mux.HandleFunc("DELETE /mcp/{alias}", p.handleMCP)
	mux.HandleFunc("POST /a2a/{alias}", p.handleA2AJSONRPC)
	mux.HandleFunc("GET /a2a/{alias}/.well-known/agent-card.json", p.handleA2AAgentCard)
	return p.withGatewayMetrics(withCORS(mux))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		allowHeaders := "Authorization, Content-Type, X-AFI-Tags, x-api-key, anthropic-version"
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

func tokenMetrics(prompt, completion int64) map[string]any {
	if prompt == 0 && completion == 0 {
		return nil
	}
	return map[string]any{
		"prompt_tokens":     prompt,
		"completion_tokens": completion,
	}
}

func (p *Pipeline) buildAttempts(snap *snapshot.Snapshot, route snapshot.Route, primary snapshot.Provider) []routeAttempt {
	cands := []routing.Candidate{{
		ProviderID: primary.ID, ProviderType: primary.Type, TargetModel: route.TargetModel, Weight: route.Weight,
	}}
	for _, fb := range route.Fallbacks {
		prov, ok := snap.Providers[fb.ProviderID]
		if !ok {
			continue
		}
		target := fb.TargetModel
		if target == "" {
			target = route.TargetModel
		}
		cands = append(cands, routing.Candidate{
			ProviderID: fb.ProviderID, ProviderType: prov.Type, TargetModel: target, Weight: fb.Weight,
		})
	}
	ordered := routing.ForStrategy(route.RoutingStrategy, p.RouteSignals).Order(cands, p.RouteRand)
	out := make([]routeAttempt, 0, len(ordered))
	for _, c := range ordered {
		prov, ok := snap.Providers[c.ProviderID]
		if !ok {
			continue
		}
		out = append(out, routeAttempt{Provider: prov, TargetModel: c.TargetModel})
	}
	return out
}

func (p *Pipeline) observeRouteAttempt(providerID, targetModel string, started time.Time, failed bool) {
	if p == nil || p.RouteSignals == nil {
		return
	}
	p.RouteSignals.Observe(providerID, targetModel, time.Since(started).Milliseconds(), failed)
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
	embedding := caps.Embedding && modelLooksLikeEmbedding(virtualModel, targetModel)
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
			chat, stream, tts, stt, embedding = false, false, caps.TTS, false, false
		case entry.IsSTT():
			chat, stream, tts, stt, embedding = false, false, false, caps.STT, false
		case entry.IsEmbedding():
			chat, stream, tts, stt, embedding = false, false, false, false, caps.Embedding
		default:
			chat = caps.Chat
			stream = caps.Stream && caps.Chat && entry.StreamingEnabled()
			tts, stt, embedding = false, false, false
		}
	} else {
		switch {
		case tts:
			mode = modelcatalog.ModeAudioSpeech
			chat, stream, stt, embedding = false, false, false, false
		case stt:
			mode = modelcatalog.ModeAudioTranscription
			chat, stream, tts, embedding = false, false, false, false
		case embedding:
			mode = modelcatalog.ModeEmbedding
			chat, stream, tts, stt = false, false, false, false
		default:
			mode = modelcatalog.ModeChat
			tts, stt, embedding = false, false, false
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
		"supports_embedding": embedding,
		"capabilities": map[string]bool{
			"chat":      chat,
			"stream":    stream,
			"tts":       tts,
			"stt":       stt,
			"embedding": embedding,
		},
	}
	if maxIn > 0 {
		item["max_input_tokens"] = maxIn
	}
	if maxOut > 0 {
		item["max_output_tokens"] = maxOut
	}
	if supportsVision && ir.DialectSupportsVision {
		item["supports_vision"] = true
	}
	if supportsTools && ir.DialectSupportsTools {
		item["supports_tools"] = true
	}
	return item
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
	if p.Metrics != nil {
		p.Metrics.RecordTokens(context.Background(), e.Modality, e.PromptTokens, e.CompletionTokens)
		if e.LatencyMs > 0 && e.ProviderType != "" {
			p.Metrics.RecordUpstream(context.Background(), e.Modality, e.ProviderType, float64(e.LatencyMs)/1000)
		}
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

// virtualAPIKey extracts the AFI virtual key from Authorization: Bearer or Anthropic-style x-api-key.
func virtualAPIKey(r *http.Request) (string, error) {
	if tok, err := bearerToken(r.Header.Get("Authorization")); err == nil {
		return tok, nil
	}
	if key := strings.TrimSpace(r.Header.Get("x-api-key")); key != "" {
		return key, nil
	}
	return "", errors.New("missing bearer")
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
