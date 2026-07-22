package llm

import (
	"context"
	"net/http"
)

type extraHeadersKey struct{}

// WithExtraHeaders attaches outbound header overlays for provider HTTP calls.
func WithExtraHeaders(ctx context.Context, headers map[string]string) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	cp := make(map[string]string, len(headers))
	for k, v := range headers {
		cp[k] = v
	}
	return context.WithValue(ctx, extraHeadersKey{}, cp)
}

func applyExtraHeaders(ctx context.Context, req *http.Request) {
	if req == nil {
		return
	}
	h, _ := ctx.Value(extraHeadersKey{}).(map[string]string)
	for k, v := range h {
		if k == "" {
			continue
		}
		req.Header.Set(http.CanonicalHeaderKey(k), v)
	}
}
