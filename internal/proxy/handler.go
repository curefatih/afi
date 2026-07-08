package proxy

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/providers"
	"github.com/curefatih/afi/internal/routing"
	"github.com/curefatih/afi/internal/types"
)

type Handler struct {
	cfg      *config.Config
	registry *providers.Registry
	hooks    *HookRunner
}

type HandlerDeps struct {
	Config   *config.Config
	Registry *providers.Registry
	Hooks    *HookRunner
}

func NewHandler(deps HandlerDeps) *Handler {
	return &Handler{
		cfg:      deps.Config,
		registry: deps.Registry,
		hooks:    deps.Hooks,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
		return
	case r.Method == http.MethodPost && r.URL.Path == "/v1/chat/completions":
		h.handleChatCompletions(w, r)
		return
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req types.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}
	reqCtx := &RequestContext{
		Model:   req.Model,
		Headers: r.Header,
		Body:    json.RawMessage(body),
	}
	h.hooks.Run(r.Context(), HookOnRequest, reqCtx)
	req.Model = reqCtx.Model

	decision, ok := h.resolveRouting(r, &req)
	if !ok {
		http.Error(w, "Failed to resolve routing", http.StatusBadRequest)
		return
	}

	provider, ok := h.registry.Get(decision.Provider)
	if !ok {
		http.Error(w, "Provider not found", http.StatusInternalServerError)
		return
	}
	reqCtx.Model = decision.Model
	// reqCtx.Metadata["_routing_decision"] = decision.Provider + "/" + decision.Model
	h.hooks.Run(r.Context(), HookOnBeforeUpstream, reqCtx)
	resp, err := provider.UpstreamChatCompletion(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	provider.WriteResponse(resp, &req, w)

	h.hooks.Run(r.Context(), HookOnResponse, reqCtx)
}

func (h *Handler) resolveRouting(r *http.Request, req *types.ChatCompletionRequest) (routing.Decision, bool) {
	return routing.NewEngine(h.cfg).Resolve(routing.Input{
		Request: req,
		Headers: r.Header,
	})
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
