package dataplane

import (
	"bytes"
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

func modelLooksLikeTTS(requested, target string) bool {
	return audioModelHint(requested, target, "tts")
}

func modelLooksLikeSTT(requested, target string) bool {
	return audioModelHint(requested, target, "stt")
}

func audioModelHint(requested, target, kind string) bool {
	for _, m := range []string{requested, target} {
		s := strings.ToLower(strings.TrimSpace(m))
		switch kind {
		case "tts":
			if strings.Contains(s, "tts") {
				return true
			}
			// ElevenLabs TTS model ids (eleven_multilingual_v2, eleven_flash_v2_5, …).
			if strings.Contains(s, "eleven_") && !strings.Contains(s, "scribe") {
				return true
			}
		case "stt":
			if strings.Contains(s, "whisper") || strings.Contains(s, "transcribe") || strings.Contains(s, "stt") || strings.Contains(s, "scribe") {
				return true
			}
		}
	}
	return false
}

func audioProviderUsable(p *Pipeline, prov snapshot.Provider, needTTS, needSTT bool) bool {
	caps := snapshot.NormalizeCapabilities(prov.Type, prov.Capabilities)
	if needTTS && !caps.TTS {
		return false
	}
	if needSTT && !caps.STT {
		return false
	}
	if p == nil || p.Providers == nil {
		return false
	}
	_, ok := p.Providers.AudioBackend(prov.Type)
	return ok
}

// buildAudioAttempts orders primary + fallbacks for TTS or STT, skipping providers
// that lack an audio backend or the required capability.
func (p *Pipeline) buildAudioAttempts(snap *snapshot.Snapshot, route snapshot.Route, primary snapshot.Provider, needTTS, needSTT bool) []routeAttempt {
	var cands []routing.Candidate
	if audioProviderUsable(p, primary, needTTS, needSTT) {
		cands = append(cands, routing.Candidate{
			ProviderID: primary.ID, ProviderType: primary.Type, TargetModel: route.TargetModel, Weight: route.Weight,
		})
	}
	for _, fb := range route.Fallbacks {
		prov, ok := snap.Providers[fb.ProviderID]
		if !ok || !audioProviderUsable(p, prov, needTTS, needSTT) {
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

func (p *Pipeline) handleAudioSpeech(w http.ResponseWriter, r *http.Request) {
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
		Model string `json:"model"`
		Input string `json:"input"`
	}
	if err := json.Unmarshal(body, &reqBody); err != nil || reqBody.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "model is required", "type": "invalid_request_error"},
		})
		return
	}
	ttsMetrics := map[string]any{}
	if n := len([]rune(reqBody.Input)); n > 0 {
		ttsMetrics["characters"] = n
	}

	call := newCallContext(key, reqBody.Model, "/v1/audio/speech", ModalityTTS, false, body, TagsFromRequest(r))
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
	caps := snapshot.NormalizeCapabilities(provider.Type, provider.Capabilities)
	if !caps.TTS || !audioProviderUsable(p, provider, true, false) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "TTS requires a provider with tts capability and an audio backend",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	if !modelLooksLikeTTS(reqBody.Model, route.TargetModel) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "model is not a TTS model (use tts-1, eleven_* , or a *tts* route)",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	attempts := p.buildAudioAttempts(snap, route, provider, true, false)
	if len(attempts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no usable providers for route", "type": "invalid_request_error"},
		})
		return
	}

	retryCfg := snap.ResolveRetry(route)
	log.Info("audio.speech",
		"model", reqBody.Model,
		"target_model", route.TargetModel,
		"provider", provider.ID,
		"fallbacks", len(route.Fallbacks),
		"retry_max_attempts", maxTriesFor(retryCfg),
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
			client, err := p.audioBackend(bound.Type)
			if err != nil {
				lastErr = err
				log.Warn("upstream audio backend missing", "provider", bound.ID, "err", err, "attempt", i)
				if i+1 < len(attempts) {
					continue targetLoop
				}
				break targetLoop
			}
			attemptStart := time.Now()
			resp, lastErr = client.AudioSpeech(llm.WithExtraHeaders(ctx, call.RequestHeaders), bound, attempt.TargetModel, body)
			if lastErr != nil {
				p.observeRouteAttempt(attempt.Provider.ID, attempt.TargetModel, attemptStart, true)
				log.Warn("upstream speech attempt failed", "provider", bound.ID, "err", lastErr, "attempt", i, "try", try)
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
			p.observeRouteAttempt(attempt.Provider.ID, attempt.TargetModel, attemptStart, resp.StatusCode >= 400)
			if shouldFailoverStatus(resp.StatusCode) {
				if try+1 < maxTries || i+1 < len(attempts) {
					log.Warn("upstream speech attempt status", "provider", bound.ID, "status", resp.StatusCode, "attempt", i, "try", try)
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
		log.Error("audio speech upstream", "err", lastErr)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": lastErr.Error(), "type": "server_error"},
		})
		status = "error"
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID, CredentialID: usedCredentialID,
			Model: reqBody.Model, ProviderType: usedProvider.Type, TargetModel: usedTarget,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalityTTS, Metrics: ttsMetrics, Tags: cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: usedProvider.Type, TargetModel: usedTarget,
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
		status = "error"
	}
	applyResponseHeaders(w, call)
	if err := CopyResponse(w, resp); err != nil {
		log.Error("copy speech response", "err", err)
		status = "error"
	}
	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID, CredentialID: usedCredentialID,
		Model: reqBody.Model, ProviderType: usedProvider.Type, TargetModel: usedTarget,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalityTTS, Metrics: ttsMetrics, Tags: cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: usedProvider.Type, TargetModel: usedTarget,
	})
}

func (p *Pipeline) handleAudioTranscriptions(w http.ResponseWriter, r *http.Request) {
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "failed to read body", "type": "invalid_request_error"},
		})
		return
	}
	ct := r.Header.Get("Content-Type")
	model, err := multipartFormValue(ct, bytes.NewReader(body), "model")
	if err != nil || model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "model is required (multipart)", "type": "invalid_request_error"},
		})
		return
	}

	call := newCallContext(key, model, "/v1/audio/transcriptions", ModalitySTT, false, body, TagsFromRequest(r))
	call.Headers = HeadersForPolicy(r.Header)
	if !p.gateCall(ctx, w, snap, call) {
		return
	}
	body = call.Body

	route, provider, ok := snap.LookupRoute(key.OrganizationID, model)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no route for model", "type": "invalid_request_error"},
		})
		return
	}
	caps := snapshot.NormalizeCapabilities(provider.Type, provider.Capabilities)
	if !caps.STT || !audioProviderUsable(p, provider, false, true) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "STT requires a provider with stt capability and an audio backend",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	if !modelLooksLikeSTT(model, route.TargetModel) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "model is not an STT model (use whisper-1, scribe_*, or a *transcribe* route, not tts-*)",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	attempts := p.buildAudioAttempts(snap, route, provider, false, true)
	if len(attempts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no usable providers for route", "type": "invalid_request_error"},
		})
		return
	}

	retryCfg := snap.ResolveRetry(route)
	log.Info("audio.transcriptions",
		"model", model,
		"target_model", route.TargetModel,
		"provider", provider.ID,
		"fallbacks", len(route.Fallbacks),
		"retry_max_attempts", maxTriesFor(retryCfg),
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
				Model:   model,
				Path:    call.Route.Path,
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
			client, err := p.audioBackend(bound.Type)
			if err != nil {
				lastErr = err
				log.Warn("upstream audio backend missing", "provider", bound.ID, "err", err, "attempt", i)
				if i+1 < len(attempts) {
					continue targetLoop
				}
				break targetLoop
			}
			attemptStart := time.Now()
			resp, lastErr = client.AudioTranscriptions(llm.WithExtraHeaders(ctx, call.RequestHeaders), bound, attempt.TargetModel, ct, bytes.NewReader(body))
			if lastErr != nil {
				p.observeRouteAttempt(attempt.Provider.ID, attempt.TargetModel, attemptStart, true)
				log.Warn("upstream transcriptions attempt failed", "provider", bound.ID, "err", lastErr, "attempt", i, "try", try)
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
			p.observeRouteAttempt(attempt.Provider.ID, attempt.TargetModel, attemptStart, resp.StatusCode >= 400)
			if shouldFailoverStatus(resp.StatusCode) {
				if try+1 < maxTries || i+1 < len(attempts) {
					log.Warn("upstream transcriptions attempt status", "provider", bound.ID, "status", resp.StatusCode, "attempt", i, "try", try)
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
		log.Error("audio transcriptions upstream", "err", lastErr)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": lastErr.Error(), "type": "server_error"},
		})
		status = "error"
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID, CredentialID: usedCredentialID,
			Model: model, ProviderType: usedProvider.Type, TargetModel: usedTarget,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalitySTT, Tags: cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: usedProvider.Type, TargetModel: usedTarget,
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
		status = "error"
	}
	applyResponseHeaders(w, call)
	if err := CopyResponse(w, resp); err != nil {
		log.Error("copy transcription response", "err", err)
		status = "error"
	}
	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID, CredentialID: usedCredentialID,
		Model: model, ProviderType: usedProvider.Type, TargetModel: usedTarget,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalitySTT, Tags: cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: usedProvider.Type, TargetModel: usedTarget,
	})
}
