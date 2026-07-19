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

func TestGeminiGenerateContentMapsContent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/models/gemini-2.0-flash:generateContent") {
			http.Error(w, "bad path "+r.URL.Path, http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("key") != "gem-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"role":  "model",
						"parts": []map[string]string{{"text": "hello from gemini"}},
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]int{
				"promptTokenCount": 3, "candidatesTokenCount": 4,
			},
		})
	}))
	defer upstream.Close()

	t.Setenv("GEMINI_API_KEY", "gem-key")
	client := NewGeminiClient()
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_gem", Type: "gemini", BaseURL: upstream.URL, APIKeyEnv: "GEMINI_API_KEY",
	}
	body := []byte(`{"model":"gemini","messages":[{"role":"user","content":"hi"}]}`)
	resp, err := client.GenerateContent(t.Context(), provider, "gemini-2.0-flash", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "hello from gemini") {
		t.Fatalf("body=%s", raw)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content != "hello from gemini" {
		t.Fatalf("unexpected: %s", raw)
	}
}
