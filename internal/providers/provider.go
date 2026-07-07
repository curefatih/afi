package providers

import (
	"context"
	"net/http"

	"github.com/curefatih/afi/internal/types"
)

// Provider translates OpenAI-compatible requests to upstream provider calls.
type Provider interface {
	Name() string
	UpstreamChatCompletion(ctx context.Context, req *types.ChatCompletionRequest) (*http.Response, error)
	WriteResponse(resp *http.Response, req *types.ChatCompletionRequest, w http.ResponseWriter) error
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(items map[string]Provider) *Registry {
	return &Registry{providers: items}
}

func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

func RetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}
