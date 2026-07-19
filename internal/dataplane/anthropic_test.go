package dataplane

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestAnthropicMessagesMapsContent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("x-api-key") != "anth-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-version") == "" {
			http.Error(w, "missing version", http.StatusBadRequest)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req["model"] != "claude-sonnet-4-20250514" {
			http.Error(w, "bad model", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "msg_test",
			"model": "claude-sonnet-4-20250514",
			"role":  "assistant",
			"content": []map[string]string{
				{"type": "text", "text": "hello from anthropic"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 5, "output_tokens": 4},
		})
	}))
	defer upstream.Close()

	t.Setenv("ANTHROPIC_API_KEY", "anth-key")

	client := NewAnthropicClient()
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_anth", Type: "anthropic", BaseURL: upstream.URL, APIKeyEnv: "ANTHROPIC_API_KEY",
	}
	body := []byte(`{"model":"claude-sonnet","messages":[{"role":"user","content":"hi"}],"max_tokens":256}`)
	resp, err := client.Messages(t.Context(), provider, "claude-sonnet-4-20250514", body, false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content != "hello from anthropic" {
		t.Fatalf("unexpected body: %s", raw)
	}
	if out.Usage.PromptTokens != 5 || out.Usage.CompletionTokens != 4 {
		t.Fatalf("usage: %+v", out.Usage)
	}
}

func TestAnthropicMessagesStreamMapsChunks(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["stream"] != true {
			http.Error(w, "expected stream", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		events := []string{
			`data: {"type":"message_start","message":{"id":"msg_s","model":"claude-x","role":"assistant"}}` + "\n\n",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hel"}}` + "\n\n",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"lo"}}` + "\n\n",
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}` + "\n\n",
			`data: {"type":"message_stop"}` + "\n\n",
		}
		for _, e := range events {
			_, _ = w.Write([]byte(e))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer upstream.Close()

	t.Setenv("ANTHROPIC_API_KEY", "anth-key")
	client := NewAnthropicClient()
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_anth", Type: "anthropic", BaseURL: upstream.URL, APIKeyEnv: "ANTHROPIC_API_KEY",
	}
	body := []byte(`{"model":"claude-x","messages":[{"role":"user","content":"hi"}],"stream":true}`)
	resp, err := client.Messages(t.Context(), provider, "claude-x", body, true)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	s := string(raw)
	if !strings.Contains(s, `"content":"hel"`) || !strings.Contains(s, `"content":"lo"`) {
		t.Fatalf("missing content chunks: %s", s)
	}
	if !strings.Contains(s, "data: [DONE]") {
		t.Fatalf("missing DONE: %s", s)
	}
	if !strings.Contains(s, "chat.completion.chunk") {
		t.Fatalf("not openai chunks: %s", s)
	}
}

func TestOpenAIChatToAnthropicExtractsSystem(t *testing.T) {
	body := []byte(`{
		"model":"x",
		"messages":[
			{"role":"system","content":"be brief"},
			{"role":"user","content":"hi"}
		],
		"max_tokens":100
	}`)
	raw, err := openAIChatToAnthropic(body, "claude-x", false)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out["system"] != "be brief" {
		t.Fatalf("system=%v", out["system"])
	}
	if out["model"] != "claude-x" {
		t.Fatalf("model=%v", out["model"])
	}
}
