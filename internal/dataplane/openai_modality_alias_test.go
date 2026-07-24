package dataplane

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestOpenAIModalityAliasPaths(t *testing.T) {
	cases := []struct {
		name         string
		aliasPath    string
		upstreamPath string
		body         string
		contentType  string
		wantModality string
		wantSnippet  string
	}{
		{
			name:         "embeddings",
			aliasPath:    "/openai/v1/embeddings",
			upstreamPath: "/v1/embeddings",
			body:         `{"model":"text-embedding-3-small","input":"hi"}`,
			contentType:  "application/json",
			wantModality: ModalityEmbedding,
			wantSnippet:  `"embedding"`,
		},
		{
			name:         "images",
			aliasPath:    "/openai/v1/images/generations",
			upstreamPath: "/v1/images/generations",
			body:         `{"model":"dall-e-3","prompt":"a cat"}`,
			contentType:  "application/json",
			wantModality: ModalityImage,
			wantSnippet:  `"b64_json"`,
		},
		{
			name:         "tts",
			aliasPath:    "/openai/v1/audio/speech",
			upstreamPath: "/v1/audio/speech",
			body:         `{"model":"tts-1","input":"hello","voice":"alloy"}`,
			contentType:  "application/json",
			wantModality: ModalityTTS,
			wantSnippet:  "AUDIO",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := modalityFromPath(tc.aliasPath); got != tc.wantModality {
				t.Fatalf("modalityFromPath(%q)=%q want %q", tc.aliasPath, got, tc.wantModality)
			}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.upstreamPath {
					t.Fatalf("upstream path=%s want %s", r.URL.Path, tc.upstreamPath)
				}
				switch tc.wantModality {
				case ModalityTTS:
					w.Header().Set("Content-Type", "audio/mpeg")
					_, _ = w.Write([]byte("AUDIO"))
				case ModalityImage:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"created":1,"data":[{"b64_json":"QQ=="}]}`))
				default:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1]}],"usage":{"prompt_tokens":1,"total_tokens":1}}`))
				}
			}))
			t.Cleanup(upstream.Close)

			t.Setenv("OPENAI_API_KEY", "sk-test")
			client := llm.NewOpenAIClient(nil)
			client.HTTP = upstream.Client()

			raw := "sk-good"
			holder := NewHolder()
			model := "text-embedding-3-small"
			caps := snapshot.DefaultCapabilities("openai")
			switch tc.wantModality {
			case ModalityImage:
				model = "dall-e-3"
			case ModalityTTS:
				model = "tts-1"
			}
			holder.Set(snapshot.Compile(snapshot.Source{
				APIKeys: []snapshot.APIKey{{
					ID: "k1", KeyHash: snapshot.HashKey(raw), ProjectID: "p1", OrganizationID: "o1",
				}},
				Providers: []snapshot.Provider{{
					ID: "prov", Type: "openai", BaseURL: upstream.URL + "/v1",
					APIKeyEnv: "OPENAI_API_KEY", Capabilities: caps,
				}},
				Routes: []snapshot.Route{{
					OrganizationID: "o1", Model: model, ProviderID: "prov", TargetModel: model,
				}},
			}))

			p := NewPipelineWithRegistry(holder, RegistryWithOpenAI(client), slog.Default())
			req := httptest.NewRequest(http.MethodPost, tc.aliasPath, bytes.NewBufferString(tc.body))
			req.Header.Set("Authorization", "Bearer "+raw)
			req.Header.Set("Content-Type", tc.contentType)
			rr := httptest.NewRecorder()
			p.Handler().ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			if !bytes.Contains(rr.Body.Bytes(), []byte(tc.wantSnippet)) {
				body, _ := io.ReadAll(rr.Body)
				t.Fatalf("body=%s", body)
			}
		})
	}
}

func TestModalityFromPathOpenAIAliases(t *testing.T) {
	pairs := []struct {
		path string
		want string
	}{
		{"/openai/v1/embeddings", ModalityEmbedding},
		{"/v1/embeddings", ModalityEmbedding},
		{"/openai/v1/images/generations", ModalityImage},
		{"/v1/images/generations", ModalityImage},
		{"/openai/v1/audio/speech", ModalityTTS},
		{"/v1/audio/speech", ModalityTTS},
		{"/openai/v1/audio/transcriptions", ModalitySTT},
		{"/v1/audio/transcriptions", ModalitySTT},
	}
	for _, p := range pairs {
		if got := modalityFromPath(p.path); got != p.want {
			t.Fatalf("%s -> %q want %q", p.path, got, p.want)
		}
	}
}
