package dataplane

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
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

	if !p.checkPolicies(w, snap, key, reqBody.Model, "/v1/messages", reqBody.Stream) {
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

	attempts := buildAnthropicAttempts(snap, route, provider)
	if len(attempts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "native anthropic path requires anthropic provider",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	client, err := p.anthropicClient()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		return
	}
	log.Info("messages",
		"project_id", key.ProjectID,
		"model", reqBody.Model,
		"provider", attempts[0].Provider.ID,
		"stream", reqBody.Stream,
		"snapshot_version", snap.Version,
	)

	var (
		resp         *http.Response
		lastErr      error
		usedProvider snapshot.Provider
		usedTarget   string
		status       = "ok"
	)

	for i, attempt := range attempts {
		usedProvider = attempt.Provider
		usedTarget = attempt.TargetModel
		resp, lastErr = client.PassThrough(ctx, attempt.Provider, attempt.TargetModel, body, reqBody.Stream)
		if lastErr != nil {
			log.Warn("upstream messages attempt failed", "provider", attempt.Provider.ID, "err", lastErr, "attempt", i)
			if i+1 < len(attempts) && shouldFailoverError(lastErr) {
				continue
			}
			break
		}
		if shouldFailoverStatus(resp.StatusCode) && i+1 < len(attempts) {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			resp = nil
			continue
		}
		break
	}

	if lastErr != nil && resp == nil {
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID,
			ProjectID:      key.ProjectID,
			APIKeyID:       key.ID,
			Model:          reqBody.Model,
			ProviderType:   usedProvider.Type,
			TargetModel:    usedTarget,
			Status:         "error",
			LatencyMs:      time.Since(start).Milliseconds(),
			Modality:       ModalityMessages,
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
		Modality:         ModalityMessages,
		Metrics:          tokenMetrics(promptTokens, completionTokens),
	})
}

func buildAnthropicAttempts(snap *snapshot.Snapshot, route snapshot.Route, primary snapshot.Provider) []routeAttempt {
	var out []routeAttempt
	if primary.Type == "anthropic" {
		out = append(out, routeAttempt{Provider: primary, TargetModel: route.TargetModel})
	}
	for _, fb := range route.Fallbacks {
		p, ok := snap.Providers[fb.ProviderID]
		if !ok || p.Type != "anthropic" {
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
