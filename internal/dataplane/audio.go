package dataplane

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
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
		case "stt":
			if strings.Contains(s, "whisper") || strings.Contains(s, "transcribe") || strings.Contains(s, "stt") {
				return true
			}
		}
	}
	return false
}

func audioOpenAICompatible(typ string) bool {
	return typ == "openai" || typ == "openai_compatible"
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

	if !p.checkPolicies(w, snap, key, reqBody.Model, "/v1/audio/speech", false) {
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
			"error": map[string]string{"message": "quota exceeded", "type": "insufficient_quota", "code": "insufficient_quota"},
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
	caps := snapshot.NormalizeCapabilities(provider.Type, provider.Capabilities)
	if !audioOpenAICompatible(provider.Type) || !caps.TTS {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "TTS requires an openai or openai_compatible provider with tts capability",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	if !modelLooksLikeTTS(reqBody.Model, route.TargetModel) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "model is not a TTS model (use tts-1 or a *tts* route)",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	log.Info("audio.speech", "model", reqBody.Model, "target_model", route.TargetModel, "provider", provider.ID)
	client, err := p.openaiAudioClient()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		return
	}
	resp, err := client.AudioSpeech(ctx, provider, route.TargetModel, body)
	status := "ok"
	if err != nil {
		log.Error("audio speech upstream", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		status = "error"
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
			Model: reqBody.Model, ProviderType: provider.Type, TargetModel: route.TargetModel,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalityTTS, Metrics: ttsMetrics,
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		status = "error"
	}
	if err := CopyResponse(w, resp); err != nil {
		log.Error("copy speech response", "err", err)
		status = "error"
	}
	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
		Model: reqBody.Model, ProviderType: provider.Type, TargetModel: route.TargetModel,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalityTTS, Metrics: ttsMetrics,
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

	if !p.checkPolicies(w, snap, key, model, "/v1/audio/transcriptions", false) {
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
			"error": map[string]string{"message": "quota exceeded", "type": "insufficient_quota", "code": "insufficient_quota"},
		})
		return
	}

	route, provider, ok := snap.LookupRoute(key.OrganizationID, model)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no route for model", "type": "invalid_request_error"},
		})
		return
	}
	caps := snapshot.NormalizeCapabilities(provider.Type, provider.Capabilities)
	if !audioOpenAICompatible(provider.Type) || !caps.STT {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "STT requires an openai or openai_compatible provider with stt capability",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	if !modelLooksLikeSTT(model, route.TargetModel) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "model is not an STT model (use whisper-1 or a *transcribe* route, not tts-*)",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	log.Info("audio.transcriptions", "model", model, "target_model", route.TargetModel, "provider", provider.ID)
	client, err := p.openaiAudioClient()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		return
	}
	resp, err := client.AudioTranscriptions(ctx, provider, route.TargetModel, ct, bytes.NewReader(body))
	status := "ok"
	if err != nil {
		log.Error("audio transcriptions upstream", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		status = "error"
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
			Model: model, ProviderType: provider.Type, TargetModel: route.TargetModel,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalitySTT,
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		status = "error"
	}
	if err := CopyResponse(w, resp); err != nil {
		log.Error("copy transcription response", "err", err)
		status = "error"
	}
	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, APIKeyID: key.ID,
		Model: model, ProviderType: provider.Type, TargetModel: route.TargetModel,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		Modality: ModalitySTT,
	})
}
