package providers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/config"
	"github.com/curefatih/afi/internal/types"
)

func TestOpenAIUpstreamChatCompletionSetsStreamAcceptHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("unexpected Accept header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[],"usage":{}}`))
	}))
	defer srv.Close()

	provider := NewOpenAI("openai", config.Provider{
		BaseURL: srv.URL,
		APIKey:  "test-key",
	})
	resp, err := provider.UpstreamChatCompletion(context.Background(), &types.ChatCompletionRequest{
		Model:  "gpt-4o-mini",
		Stream: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
}

func TestCopyUpstreamResponseFlushesSSE(t *testing.T) {
	rec := httptest.NewRecorder()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: hello\n\n")),
	}

	if err := copyUpstreamResponse(rec, resp); err != nil {
		t.Fatal(err)
	}
	if !rec.Flushed {
		t.Fatal("expected SSE response to flush")
	}
	if got := rec.Body.String(); got != "data: hello\n\n" {
		t.Fatalf("unexpected body: %q", got)
	}
}
