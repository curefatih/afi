package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestOpenAIChatCompletions(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl_test",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "hi"}},
			},
		})
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "sk-test")
	client := NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_oai", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
	}
	body := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`)
	resp, err := client.ChatCompletions(t.Context(), provider, "gpt-4o-mini", body, false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotBody["model"] != "gpt-4o-mini" {
		t.Fatalf("model=%v", gotBody["model"])
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "chatcmpl_test") {
		t.Fatalf("body=%s", raw)
	}
}

func TestOpenAIChatCompletionsStreamOptions(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	client := NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_oai", Type: "openai", BaseURL: upstream.URL, InlineAPIKey: "sk-inline",
	}
	resp, err := client.ChatCompletions(t.Context(), provider, "gpt-4o", []byte(`{"messages":[]}`), true)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if gotBody["stream"] != true {
		t.Fatalf("stream=%v", gotBody["stream"])
	}
	opts, _ := gotBody["stream_options"].(map[string]any)
	if opts["include_usage"] != true {
		t.Fatalf("stream_options=%v", gotBody["stream_options"])
	}
}

func TestOpenAIMissingAPIKey(t *testing.T) {
	client := NewOpenAIClient(nil)
	provider := snapshot.Provider{
		ID: "prov_oai", Type: "openai", BaseURL: "http://example.invalid", APIKeyEnv: "AFI_MISSING_OPENAI_KEY",
	}
	_, err := client.ChatCompletions(t.Context(), provider, "gpt-4o", []byte(`{"messages":[]}`), false)
	if err == nil {
		t.Fatal("expected missing key error")
	}
}
