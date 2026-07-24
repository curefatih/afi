package dataplane

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func (p *Pipeline) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	p.executeChat(w, r, ir.DialectOpenAI, ModalityChat, chatExecOptions{})
}

func (p *Pipeline) handleMessages(w http.ResponseWriter, r *http.Request) {
	p.executeChat(w, r, ir.DialectAnthropic, ModalityMessages, chatExecOptions{})
}

type chatExecOptions struct {
	model  string
	stream bool
}

func (p *Pipeline) handleGeminiGenerateContent(w http.ResponseWriter, r *http.Request) {
	operation := r.PathValue("operation")
	model, stream, ok := parseGeminiOperation(operation)
	if !ok {
		dialect.WriteError(
			w, ir.DialectGemini, http.StatusNotFound,
			fmt.Sprintf("unsupported Gemini operation %q", operation), "NOT_FOUND",
		)
		return
	}
	if model == "" {
		dialect.WriteError(w, ir.DialectGemini, http.StatusBadRequest, "model route is required", "INVALID_ARGUMENT")
		return
	}
	p.executeChat(w, r, ir.DialectGemini, ModalityGenerateContent, chatExecOptions{
		model: model, stream: stream,
	})
}

// parseGeminiOperation splits "{route}:generateContent" / "{route}:streamGenerateContent"
// using a suffix match so route aliases may themselves contain ":".
func parseGeminiOperation(operation string) (model string, stream bool, ok bool) {
	if model, ok := strings.CutSuffix(operation, ":streamGenerateContent"); ok {
		return model, true, true
	}
	if model, ok := strings.CutSuffix(operation, ":generateContent"); ok {
		return model, false, true
	}
	return "", false, false
}

func (p *Pipeline) executeChat(
	w http.ResponseWriter,
	r *http.Request,
	d ir.Dialect,
	modality string,
	opts chatExecOptions,
) {
	reqID := kernel.NewRequestID()
	ctx := kernel.WithRequestID(r.Context(), reqID)
	log := p.Log.With("request_id", reqID)
	start := time.Now()

	codec, err := dialect.For(d)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	rawKey, err := virtualAPIKey(r)
	if err != nil {
		dialect.WriteError(w, d, http.StatusUnauthorized, "missing or invalid authorization", "invalid_request_error")
		return
	}

	snap := p.Holder.Get()
	if snap == nil {
		dialect.WriteError(w, d, http.StatusServiceUnavailable, "no snapshot loaded", "server_error")
		return
	}

	key, ok := snap.LookupKey(rawKey)
	if !ok {
		dialect.WriteError(w, d, http.StatusUnauthorized, "invalid api key", "invalid_request_error")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		dialect.WriteError(w, d, http.StatusBadRequest, "failed to read body", "invalid_request_error")
		return
	}

	chatReq, err := codec.DecodeRequest(body)
	if err != nil {
		dialect.WriteError(w, d, http.StatusBadRequest, err.Error(), "invalid_request_error")
		return
	}
	if opts.model != "" {
		chatReq.Model = opts.model
	}
	if opts.stream {
		chatReq.Stream = true
	}

	openaiBody, err := ir.EncodeOpenAI(chatReq)
	if err != nil {
		dialect.WriteError(w, d, http.StatusBadRequest, err.Error(), "invalid_request_error")
		return
	}

	path := r.URL.Path
	call := newCallContext(key, chatReq.Model, path, modality, chatReq.Stream, openaiBody, TagsFromRequest(r))
	call.Headers = HeadersForPolicy(r.Header)
	if call.Metadata == nil {
		call.Metadata = map[string]any{}
	}
	call.Metadata["dialect"] = string(d)
	// Original client wire (OpenAI or Anthropic JSON) for BeforeCall inspection.
	// BeforeChat still mutates the OpenAI-shaped bridge body in call.Body.
	call.Metadata["client_body"] = append([]byte(nil), body...)
	if !p.gateCall(ctx, w, snap, call) {
		return
	}
	openaiBody = call.Body

	route, provider, ok := snap.LookupRoute(key.OrganizationID, chatReq.Model)
	if !ok {
		dialect.WriteError(w, d, http.StatusBadRequest, "no route for model", "invalid_request_error")
		return
	}

	attempts := p.buildAttempts(snap, route, provider)
	if len(attempts) == 0 {
		dialect.WriteError(w, d, http.StatusBadRequest, "no usable providers for route", "invalid_request_error")
		return
	}

	openaiBody, err = p.Hooks.RunBeforeChat(ctx, openaiBody)
	if err != nil {
		log.Error("chat hook", "err", err)
		dialect.WriteError(w, d, http.StatusBadRequest, "chat hook failed: "+err.Error(), "invalid_request_error")
		return
	}
	if p.Wasm != nil {
		openaiBody, err = p.Wasm.RunBeforeChat(ctx, snap, call.Principal.OrganizationID, openaiBody)
		if err != nil {
			log.Error("wasm before_chat", "err", err)
			dialect.WriteError(w, d, http.StatusBadRequest, "chat hook failed: "+err.Error(), "invalid_request_error")
			return
		}
	}
	call.Body = openaiBody

	chatReq, err = ir.DecodeOpenAIRequest(openaiBody)
	if err != nil {
		dialect.WriteError(w, d, http.StatusBadRequest, "chat hook produced invalid body: "+err.Error(), "invalid_request_error")
		return
	}
	chatReq, err = p.Hooks.RunBeforeChatIR(ctx, chatReq)
	if err != nil {
		log.Error("typed chat hook", "err", err)
		dialect.WriteError(w, d, http.StatusBadRequest, "chat hook failed: "+err.Error(), "invalid_request_error")
		return
	}
	if p.Wasm != nil {
		chatReq, err = p.Wasm.RunBeforeChatIR(ctx, snap, call.Principal.OrganizationID, chatReq)
		if err != nil {
			log.Error("wasm before_chat_ir", "err", err)
			dialect.WriteError(w, d, http.StatusBadRequest, "chat hook failed: "+err.Error(), "invalid_request_error")
			return
		}
	}
	// Preserve client-requested model for routing/usage; hooks may rewrite messages only.
	chatReq.Model = call.Route.Model
	chatReq.Stream = call.Route.Stream

	retryCfg := snap.ResolveRetry(route)
	log.Info("chat",
		"dialect", string(d),
		"project_id", key.ProjectID,
		"model", chatReq.Model,
		"target_model", route.TargetModel,
		"provider", provider.ID,
		"fallbacks", len(route.Fallbacks),
		"retry_max_attempts", maxTriesFor(retryCfg),
		"stream", chatReq.Stream,
		"snapshot_version", snap.Version,
	)

	var (
		result           ir.ChatResult
		lastErr          error
		usedProvider     snapshot.Provider
		usedTarget       string
		usedCredentialID string
		promptTokens     int64
		completionTokens int64
		status           = "ok"
		haveResult       bool
	)

	maxTries := maxTriesFor(retryCfg)
targetLoop:
	for i, attempt := range attempts {
		for try := 0; try < maxTries; try++ {
			if try > 0 {
				if sleepErr := sleepBeforeRetry(ctx, retryCfg, try-1); sleepErr != nil {
					lastErr = sleepErr
					break targetLoop
				}
			}

			bound, credID, bindErr := p.bindProviderSecret(ctx, snap, attempt.Provider, key, policy.Request{
				Model:   chatReq.Model,
				Path:    call.Route.Path,
				Stream:  chatReq.Stream,
				Tags:    call.Tags,
				Headers: call.Headers,
			})
			if bindErr != nil {
				lastErr = bindErr
				log.Warn("credential resolve failed", "provider", attempt.Provider.ID, "err", bindErr, "attempt", i)
				if i+1 < len(attempts) {
					continue targetLoop
				}
				break targetLoop
			}
			usedProvider = bound
			usedTarget = attempt.TargetModel
			usedCredentialID = credID
			attemptStart := time.Now()
			result, lastErr = p.callProviderIR(ctx, bound, attempt.TargetModel, chatReq, call)
			if lastErr != nil {
				p.observeRouteAttempt(attempt.Provider.ID, attempt.TargetModel, attemptStart, true)
				log.Warn("upstream attempt failed", "provider", attempt.Provider.ID, "err", lastErr, "attempt", i, "try", try)
				if errors.Is(lastErr, ErrStreamUnsupported) {
					break targetLoop
				}
				if _, ok := ir.AsUnsupported(lastErr); ok {
					// Feature/provider mismatch is a client error; do not fail over.
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
			haveResult = true
			failed := result.StatusCode >= 400
			p.observeRouteAttempt(attempt.Provider.ID, attempt.TargetModel, attemptStart, failed)
			if shouldFailoverStatus(result.StatusCode) {
				if try+1 < maxTries || i+1 < len(attempts) {
					log.Warn("upstream attempt status", "provider", attempt.Provider.ID, "status", result.StatusCode, "attempt", i, "try", try)
					if try+1 < maxTries {
						logRetry(log, attempt.Provider.ID, try, maxTries, result.StatusCode, nil)
						continue
					}
					continue targetLoop
				}
			}
			break targetLoop
		}
	}

	if lastErr != nil && !haveResult {
		if errors.Is(lastErr, ErrStreamUnsupported) {
			dialect.WriteError(w, d, http.StatusBadRequest, lastErr.Error(), "invalid_request_error")
			return
		}
		if u, ok := ir.AsUnsupported(lastErr); ok {
			dialect.WriteError(w, d, http.StatusBadRequest, u.Error(), "invalid_request_error")
			return
		}
		log.Error("upstream error", "err", lastErr)
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID,
			CredentialID: usedCredentialID,
			Model:        chatReq.Model,
			ProviderType: usedProvider.Type,
			TargetModel:  usedTarget,
			Status:       "error",
			LatencyMs:    time.Since(start).Milliseconds(),
			Modality:     modality,
			Tags:         cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: "error", LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: usedProvider.Type, TargetModel: usedTarget,
		})
		p.runAfterChat(ctx, d, modality, AfterChatInfo{
			Model: chatReq.Model, Status: "error",
			LatencyMs:    time.Since(start).Milliseconds(),
			ProviderType: usedProvider.Type, TargetModel: usedTarget,
		})
		dialect.WriteError(w, d, http.StatusBadGateway, lastErr.Error(), "server_error")
		return
	}
	if !haveResult {
		dialect.WriteError(w, d, http.StatusBadGateway, "all upstream attempts failed", "server_error")
		return
	}

	if result.StatusCode >= 400 {
		status = "upstream_error"
		applyResponseHeaders(w, call)
		dialect.WriteUpstreamError(w, d, result.StatusCode, result.ErrorBody)
	} else if chatReq.Stream {
		applyResponseHeaders(w, call)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		var copyErr error
		promptTokens, completionTokens, copyErr = codec.WriteStream(w, result.Events)
		if copyErr != nil {
			log.Error("copy stream response", "err", copyErr)
			status = "error"
		}
		if promptTokens+completionTokens > 0 {
			p.incrTokens(ctx, snap, key, promptTokens+completionTokens)
		}
	} else {
		if result.Response == nil {
			dialect.WriteError(w, d, http.StatusBadGateway, "empty upstream response", "server_error")
			return
		}
		// Prefer client-requested model alias in response when upstream echoes target.
		if result.Response.Model == "" {
			result.Response.Model = usedTarget
		}
		encoded, encErr := codec.EncodeResponse(*result.Response)
		if encErr != nil {
			dialect.WriteError(w, d, http.StatusBadGateway, encErr.Error(), "server_error")
			return
		}
		promptTokens = result.Response.Usage.PromptTokens
		completionTokens = result.Response.Usage.CompletionTokens
		p.incrTokens(ctx, snap, key, promptTokens+completionTokens)
		applyResponseHeaders(w, call)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(encoded)
	}

	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID,
		CredentialID:     usedCredentialID,
		Model:            chatReq.Model,
		ProviderType:     usedProvider.Type,
		TargetModel:      usedTarget,
		Status:           status,
		LatencyMs:        time.Since(start).Milliseconds(),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Modality:         modality,
		Metrics:          tokenMetrics(promptTokens, completionTokens),
		Tags:             cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: usedProvider.Type, TargetModel: usedTarget,
		PromptTokens: promptTokens, CompletionTokens: completionTokens,
	})
	p.runAfterChat(ctx, d, modality, AfterChatInfo{
		Model: chatReq.Model, Status: status,
		LatencyMs:    time.Since(start).Milliseconds(),
		ProviderType: usedProvider.Type, TargetModel: usedTarget,
	})
}

func (p *Pipeline) runAfterChat(ctx context.Context, d ir.Dialect, modality string, info AfterChatInfo) {
	if p == nil || p.Hooks == nil {
		return
	}
	info.Dialect = string(d)
	info.Modality = modality
	p.Hooks.RunAfterChat(ctx, info)
}

func (p *Pipeline) callProviderIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest, call *CallContext) (ir.ChatResult, error) {
	if p.Providers == nil {
		return ir.ChatResult{}, errors.New("provider registry not configured")
	}
	adapter, ok := p.Providers.Get(provider.Type)
	if !ok {
		return ir.ChatResult{}, errors.New("unsupported provider type \"" + provider.Type + "\"")
	}
	caps := snapshot.NormalizeCapabilities(provider.Type, provider.Capabilities)
	if req.Stream && !caps.Stream {
		return ir.ChatResult{}, ErrStreamUnsupported
	}
	if !caps.Chat {
		return ir.ChatResult{}, errors.New("chat is not supported for provider type \"" + provider.Type + "\"")
	}
	if call != nil {
		ctx = llm.WithExtraHeaders(ctx, call.RequestHeaders)
	}
	if irp, ok := adapter.(IRChatProvider); ok {
		return irp.ChatIR(ctx, provider, targetModel, req)
	}
	// Legacy SDK / gRPC adapters: OpenAI-shaped bytes bridge.
	body, err := ir.EncodeOpenAI(req)
	if err != nil {
		return ir.ChatResult{}, err
	}
	resp, err := adapter.Chat(ctx, provider, targetModel, body, req.Stream)
	if err != nil {
		return ir.ChatResult{}, err
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, ErrorBody: raw}, nil
	}
	if req.Stream {
		ch := dialect.ParseOpenAISSE(resp.Body)
		out := make(chan ir.StreamEvent, 16)
		go func() {
			defer close(out)
			defer resp.Body.Close()
			for ev := range ch {
				out <- ev
			}
		}()
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Events: out}, nil
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return ir.ChatResult{}, err
	}
	mapped, err := ir.DecodeOpenAIResponse(raw)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Response: &mapped}, nil
}
