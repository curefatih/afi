package dataplane

import (
	"encoding/json"
	"github.com/curefatih/afi/internal/adapters/llm"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestListModelsFromRoutes(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://example.invalid", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{
			{OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini"},
			{OrganizationID: "o1", Model: "tts-1", ProviderID: "prov", TargetModel: "tts-1"},
			{OrganizationID: "o1", Model: "claude-x", ProviderID: "prov", TargetModel: "claude-x"},
			{OrganizationID: "o2", Model: "other-org", ProviderID: "prov", TargetModel: "x"},
		},
	}))

	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out struct {
		Object string `json:"object"`
		Data   []struct {
			ID                string `json:"id"`
			Mode              string `json:"mode"`
			SupportsStreaming bool   `json:"supports_streaming"`
			SupportsTTS       bool   `json:"supports_tts"`
			SupportsSTT       bool   `json:"supports_stt"`
			MaxInputTokens    int    `json:"max_input_tokens"`
			Capabilities      struct {
				Chat bool `json:"chat"`
				TTS  bool `json:"tts"`
			} `json:"capabilities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Object != "list" {
		t.Fatalf("object=%s", out.Object)
	}
	byID := map[string](struct {
		ID                string `json:"id"`
		Mode              string `json:"mode"`
		SupportsStreaming bool   `json:"supports_streaming"`
		SupportsTTS       bool   `json:"supports_tts"`
		SupportsSTT       bool   `json:"supports_stt"`
		MaxInputTokens    int    `json:"max_input_tokens"`
		Capabilities      struct {
			Chat bool `json:"chat"`
			TTS  bool `json:"tts"`
		} `json:"capabilities"`
	}){}
	for _, m := range out.Data {
		byID[m.ID] = m
	}
	chat := byID["gpt-4o-mini"]
	if chat.Mode != "chat" || !chat.Capabilities.Chat || !chat.SupportsStreaming || chat.MaxInputTokens != 128000 {
		t.Fatalf("gpt-4o-mini=%+v", chat)
	}
	tts := byID["tts-1"]
	if tts.Mode != "audio_speech" || tts.Capabilities.Chat || !tts.SupportsTTS || tts.SupportsStreaming {
		t.Fatalf("tts-1=%+v", tts)
	}
	if _, ok := byID["claude-x"]; !ok {
		t.Fatalf("missing models: %+v", out.Data)
	}
	if _, ok := byID["other-org"]; ok {
		t.Fatal("leaked other org model")
	}
}

func TestListModelsUnauthorized(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
	}))
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-bad")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}
