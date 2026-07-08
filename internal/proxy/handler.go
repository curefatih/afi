package proxy

import (
	"net/http"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/providers"
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

}
