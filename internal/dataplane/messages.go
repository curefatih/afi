package dataplane

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/dataplane/routing"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func (p *Pipeline) handleMessages(w http.ResponseWriter, r *http.Request) {
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

	call := newCallContext(key, reqBody.Model, "/v1/messages", ModalityMessages, reqBody.Stream, body, TagsFromRequest(r))
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

	attempts := p.buildAnthropicAttempts(snap, route, provider)
	if len(attempts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "native anthropic path requires anthropic provider",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	retryCfg := snap.ResolveRetry(route)
	log.Info("messages",
		"project_id", key.ProjectID,
		"model", reqBody.Model,
		"provider", attempts[0].Provider.ID,
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
				if i+1 < len(attempts) {
					continue targetLoop
				}
				break targetLoop
			}
			usedProvider = bound
			usedTarget = attempt.TargetModel
			usedCredentialID = credID
			client, err := p.messagesBackend(bound.Type)
			if err != nil {
				lastErr = err
				log.Warn("upstream messages backend missing", "provider", bound.ID, "err", err, "attempt", i)
				if i+1 < len(attempts) {
					continue targetLoop
				}
				break targetLoop
			}
			resp, lastErr = client.PassThrough(llm.WithExtraHeaders(ctx, call.RequestHeaders), bound, attempt.TargetModel, body, reqBody.Stream)
			if lastErr != nil {
				log.Warn("upstream messages attempt failed", "provider", bound.ID, "err", lastErr, "attempt", i, "try", try)
				if shouldFailoverError(lastErr) {
					if try+1 < maxTries {
						logRetry(log, bound.ID, try, maxTries, 0, lastErr)
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
					code := resp.StatusCode
					discardResponse(resp)
					resp = nil
					if try+1 < maxTries {
						logRetry(log, bound.ID, try, maxTries, code, nil)
						continue
					}
					continue targetLoop
				}
			}
			break targetLoop
		}
	}

	if lastErr != nil && resp == nil {
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
			Modality:       ModalityMessages,
			Tags:           cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: "error", LatencyMs: time.Since(start).Milliseconds(),
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

	var promptTokens, completionTokens int64
	if !reqBody.Stream && resp.StatusCode < 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": map[string]string{"message": "failed to read upstream", "type": "server_error"},
			})
			return
		}
		promptTokens, completionTokens = parseAnthropicUsageTokens(respBody)
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
	} else {
		applyResponseHeaders(w, call)
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
		Modality:         ModalityMessages,
		Metrics:          tokenMetrics(promptTokens, completionTokens),
		Tags:             cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: usedProvider.Type, TargetModel: usedTarget,
		PromptTokens: promptTokens, CompletionTokens: completionTokens,
	})
}

func (p *Pipeline) buildAnthropicAttempts(snap *snapshot.Snapshot, route snapshot.Route, primary snapshot.Provider) []routeAttempt {
	var cands []routing.Candidate
	if primary.Type == "anthropic" {
		cands = append(cands, routing.Candidate{
			ProviderID: primary.ID, TargetModel: route.TargetModel, Weight: route.Weight,
		})
	}
	for _, fb := range route.Fallbacks {
		prov, ok := snap.Providers[fb.ProviderID]
		if !ok || prov.Type != "anthropic" {
			continue
		}
		target := fb.TargetModel
		if target == "" {
			target = route.TargetModel
		}
		cands = append(cands, routing.Candidate{
			ProviderID: fb.ProviderID, TargetModel: target, Weight: fb.Weight,
		})
	}
	ordered := routing.ForStrategy(route.RoutingStrategy).Order(cands, p.RouteRand)
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

func parseAnthropicUsageTokens(body []byte) (prompt, completion int64) {
	var parsed struct {
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0
	}
	return parsed.Usage.InputTokens, parsed.Usage.OutputTokens
}
