package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/providers"
	"github.com/curefatih/afi/internal/types"
)

type captureProvider struct {
	lastReq *types.ChatCompletionRequest
}

func (p *captureProvider) Name() string { return "capture" }

func (p *captureProvider) UpstreamChatCompletion(_ context.Context, req *types.ChatCompletionRequest) (*http.Response, error) {
	p.lastReq = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{\"choices\":[{\"message\":{\"role\":\"assistant\",\"content\":\"Hello, world!\"}}]}")),
	}, nil
}

func (p *captureProvider) WriteResponse(resp *http.Response, req *types.ChatCompletionRequest, w http.ResponseWriter) error {
	return nil
}

func TestHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Addr: ":8080",
		},
	}
	registry := providers.NewRegistry(map[string]providers.Provider{
		"provider1": &captureProvider{lastReq: &types.ChatCompletionRequest{Model: "model1", Messages: []types.ChatMessage{{Role: "user", Content: json.RawMessage(`{"Hello, world!"}`)}}}},
		"provider2": &captureProvider{lastReq: &types.ChatCompletionRequest{Model: "model2", Messages: []types.ChatMessage{{Role: "user", Content: json.RawMessage(`{"Hello, world!"}`)}}}},
	})
	handler := NewHandler(cfg, registry)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
}
