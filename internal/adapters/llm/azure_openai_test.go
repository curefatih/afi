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

func TestAzureEndpointURLDeployments(t *testing.T) {
	t.Parallel()
	p := snapshot.Provider{
		BaseURL: "https://myres.openai.azure.com",
		Config:  json.RawMessage(`{"api_style":"deployments"}`),
	}
	got, err := azureEndpointURL(p, "gpt-4o", "chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "https://myres.openai.azure.com/openai/deployments/gpt-4o/chat/completions?") {
		t.Fatalf("url=%q", got)
	}
	if !strings.Contains(got, "api-version="+azureDefaultAPIVersion) {
		t.Fatalf("missing default api-version: %q", got)
	}
}

func TestAzureEndpointURLDeploymentsExplicitVersion(t *testing.T) {
	t.Parallel()
	p := snapshot.Provider{
		BaseURL: "https://myres.openai.azure.com/",
		Config:  json.RawMessage(`{"api_style":"deployments","api_version":"2024-06-01"}`),
	}
	got, err := azureEndpointURL(p, "dep-1", "embeddings")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/openai/deployments/dep-1/embeddings?") {
		t.Fatalf("url=%q", got)
	}
	if !strings.Contains(got, "api-version=2024-06-01") {
		t.Fatalf("url=%q", got)
	}
}

func TestAzureEndpointURLOpenAIV1(t *testing.T) {
	t.Parallel()
	p := snapshot.Provider{
		BaseURL: "https://myres.openai.azure.com/openai/v1",
		Config:  json.RawMessage(`{"api_style":"openai_v1"}`),
	}
	got, err := azureEndpointURL(p, "ignored", "chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://myres.openai.azure.com/openai/v1/chat/completions" {
		t.Fatalf("url=%q", got)
	}
}

func TestAzureEndpointURLOpenAIV1WithVersion(t *testing.T) {
	t.Parallel()
	p := snapshot.Provider{
		BaseURL: "https://myres.openai.azure.com/openai/v1",
		Config:  json.RawMessage(`{"api_style":"openai_v1","api_version":"preview"}`),
	}
	got, err := azureEndpointURL(p, "dep", "embeddings")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://myres.openai.azure.com/openai/v1/embeddings?api-version=preview" {
		t.Fatalf("url=%q", got)
	}
}

func TestAzureEndpointURLDefaultStyle(t *testing.T) {
	t.Parallel()
	p := snapshot.Provider{BaseURL: "https://x.openai.azure.com"}
	got, err := azureEndpointURL(p, "m", "chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/openai/deployments/m/chat/completions") {
		t.Fatalf("default should be deployments: %q", got)
	}
}

func TestParseAzureConfigInvalidStyle(t *testing.T) {
	t.Parallel()
	_, err := parseAzureConfig(json.RawMessage(`{"api_style":"legacy"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAzureChatCompletionsDeployments(t *testing.T) {
	var gotPath, gotAuth, gotAPIKey string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("api-key")
		if r.URL.Query().Get("api-version") != azureDefaultAPIVersion {
			http.Error(w, "bad version", http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl_az",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "hi"}},
			},
		})
	}))
	defer upstream.Close()

	t.Setenv("AZURE_OPENAI_API_KEY", "az-test-key")
	client := NewAzureOpenAIClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_az", Type: "azure_openai",
		BaseURL:   upstream.URL,
		APIKeyEnv: "AZURE_OPENAI_API_KEY",
		Config:    json.RawMessage(`{"api_style":"deployments"}`),
	}
	body := []byte(`{"model":"ignored","messages":[{"role":"user","content":"hi"}]}`)
	resp, err := client.ChatCompletions(t.Context(), provider, "my-deploy", body, false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if gotPath != "/openai/deployments/my-deploy/chat/completions" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotAuth != "" {
		t.Fatalf("unexpected Authorization=%q", gotAuth)
	}
	if gotAPIKey != "az-test-key" {
		t.Fatalf("api-key=%q", gotAPIKey)
	}
	if gotBody["model"] != "my-deploy" {
		t.Fatalf("model=%v", gotBody["model"])
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "chatcmpl_az") {
		t.Fatalf("body=%s", raw)
	}
}

func TestAzureChatCompletionsOpenAIV1(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Header.Get("api-key") != "inline-key" {
			http.Error(w, "bad key", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "ok", "choices": []any{}})
	}))
	defer upstream.Close()

	client := NewAzureOpenAIClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_az", Type: "azure_openai",
		BaseURL:      upstream.URL + "/openai/v1",
		InlineAPIKey: "inline-key",
		Config:       json.RawMessage(`{"api_style":"openai_v1"}`),
	}
	resp, err := client.ChatCompletions(t.Context(), provider, "dep", []byte(`{"messages":[]}`), false)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if gotPath != "/openai/v1/chat/completions" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestAzureChatCompletionsStreamOptions(t *testing.T) {
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	client := NewAzureOpenAIClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		BaseURL: upstream.URL + "/openai/v1", InlineAPIKey: "k",
		Config: json.RawMessage(`{"api_style":"openai_v1"}`),
	}
	resp, err := client.ChatCompletions(t.Context(), provider, "dep", []byte(`{"messages":[]}`), true)
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

func TestAzureMissingAPIKey(t *testing.T) {
	client := NewAzureOpenAIClient(nil)
	provider := snapshot.Provider{
		ID: "prov_az", Type: "azure_openai",
		BaseURL: "http://example.invalid", APIKeyEnv: "AFI_MISSING_AZURE_KEY",
		Config: json.RawMessage(`{"api_style":"openai_v1"}`),
	}
	_, err := client.ChatCompletions(t.Context(), provider, "dep", []byte(`{"messages":[]}`), false)
	if err == nil {
		t.Fatal("expected missing key error")
	}
}
