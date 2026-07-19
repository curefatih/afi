package dataplane

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

type Pipeline struct {
	Holder *Holder
	OpenAI *OpenAIClient
	Log    *slog.Logger
}

func NewPipeline(holder *Holder, openai *OpenAIClient, log *slog.Logger) *Pipeline {
	return &Pipeline{Holder: holder, OpenAI: openai, Log: log}
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

func (p *Pipeline) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	reqID := kernel.NewRequestID()
	ctx := kernel.WithRequestID(r.Context(), reqID)
	log := p.Log.With("request_id", reqID)

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

	route, provider, ok := snap.LookupRoute(reqBody.Model)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "no route for model", "type": "invalid_request_error"},
		})
		return
	}

	if provider.Type != "openai" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]string{"message": "unsupported provider type", "type": "invalid_request_error"},
		})
		return
	}

	log.Info("chat.completions",
		"project_id", key.ProjectID,
		"model", reqBody.Model,
		"target_model", route.TargetModel,
		"provider", provider.ID,
		"stream", reqBody.Stream,
		"snapshot_version", snap.Version,
	)

	resp, err := p.OpenAI.ChatCompletions(ctx, provider, route.TargetModel, body, reqBody.Stream)
	if err != nil {
		log.Error("upstream error", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "server_error"},
		})
		return
	}
	defer resp.Body.Close()

	if err := CopyResponse(w, resp); err != nil {
		log.Error("copy response", "err", err)
	}
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
