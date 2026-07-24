package dataplane

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/dataplane/openaichat"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func modelLooksLikeEmbedding(requested, target string) bool {
	for _, m := range []string{requested, target} {
		s := strings.ToLower(strings.TrimSpace(m))
		if strings.Contains(s, "embedding") || strings.Contains(s, "embed") {
			return true
		}
	}
	return false
}

func embeddingsOpenAICompatible(typ string) bool {
	return typ == "openai" || typ == "openai_compatible"
}

func (p *Pipeline) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
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
		Input any    `json:"input"`
	}
	if err := json.Unmarshal(body, &reqBody); err != nil || reqBody.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "model is required", "type": "invalid_request_error"},
		})
		return
	}
	if reqBody.Input == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "input is required", "type": "invalid_request_error"},
		})
		return
	}

	call := newCallContext(key, reqBody.Model, "/v1/embeddings", ModalityEmbedding, false, body, TagsFromRequest(r))
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
	if !embeddingsOpenAICompatible(provider.Type) || !caps.Embedding {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "embeddings require an openai or openai_compatible provider with embedding capability",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	if !modelLooksLikeEmbedding(reqBody.Model, route.TargetModel) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{
				"message": "model is not an embedding model (use text-embedding-* or a *embed* route)",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	log.Info("embeddings", "model", reqBody.Model, "target_model", route.TargetModel, "provider", provider.ID)
	bound, credID, bindErr := p.bindProviderSecret(ctx, snap, provider, key, policy.Request{
		Model:   reqBody.Model,
		Path:    call.Route.Path,
		Tags:    call.Tags,
		Headers: call.Headers,
	})
	if bindErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": bindErr.Error(), "type": "server_error"},
		})
		return
	}
	client, err := p.embeddingsBackend(bound.Type)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		return
	}
	resp, err := client.Embeddings(llm.WithExtraHeaders(ctx, call.RequestHeaders), bound, route.TargetModel, body)
	status := "ok"
	var promptTokens, completionTokens int64
	if err != nil {
		log.Error("embeddings upstream", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		status = "error"
		p.recordUsage(UsageEvent{
			OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID, CredentialID: credID,
			Model: reqBody.Model, ProviderType: bound.Type, TargetModel: route.TargetModel,
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			Modality: ModalityEmbedding, Tags: cloneTags(call.Tags),
		})
		p.runAfterCall(ctx, snap, call, AfterCallInfo{
			Status: status, LatencyMs: time.Since(start).Milliseconds(),
			ProviderType: bound.Type, TargetModel: route.TargetModel,
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		status = "error"
	}

	if resp.StatusCode < 400 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": map[string]string{"message": "failed to read upstream", "type": "server_error"},
			})
			status = "error"
			p.recordUsage(UsageEvent{
				OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID, CredentialID: credID,
				Model: reqBody.Model, ProviderType: bound.Type, TargetModel: route.TargetModel,
				Status: status, LatencyMs: time.Since(start).Milliseconds(),
				Modality: ModalityEmbedding, Tags: cloneTags(call.Tags),
			})
			p.runAfterCall(ctx, snap, call, AfterCallInfo{
				Status: status, LatencyMs: time.Since(start).Milliseconds(),
				ProviderType: bound.Type, TargetModel: route.TargetModel,
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
	} else {
		applyResponseHeaders(w, call)
		if copyErr := CopyResponse(w, resp); copyErr != nil {
			log.Error("copy embeddings response", "err", copyErr)
			status = "error"
		}
	}

	p.recordUsage(UsageEvent{
		OrganizationID: key.OrganizationID, ProjectID: key.ProjectID, TeamID: key.TeamID, EnvironmentID: key.EnvironmentID, APIKeyID: key.ID, CredentialID: credID,
		Model: reqBody.Model, ProviderType: bound.Type, TargetModel: route.TargetModel,
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		PromptTokens: promptTokens, CompletionTokens: completionTokens,
		Modality: ModalityEmbedding, Tags: cloneTags(call.Tags),
	})
	p.runAfterCall(ctx, snap, call, AfterCallInfo{
		Status: status, LatencyMs: time.Since(start).Milliseconds(),
		ProviderType: bound.Type, TargetModel: route.TargetModel,
	})
}
