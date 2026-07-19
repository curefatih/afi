package dataplane

import (
	"github.com/curefatih/afi/internal/adapters/llm"
	"bytes"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestAudioSpeechPassThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte(`"model":"tts-1"`)) {
			t.Fatalf("body=%s", body)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3fakeaudio"))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("OPENAI_API_KEY", "sk-test")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()

	raw := "sk-audio-test"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-audio",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: upstream.URL + "/v1",
			APIKeyEnv: "OPENAI_API_KEY", Name: "OpenAI",
			Capabilities: snapshot.DefaultCapabilities("openai"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_openai", TargetModel: "tts-1",
		}},
	}))

	p := NewPipelineWithRegistry(holder, RegistryWithOpenAI(client), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", bytes.NewBufferString(
		`{"model":"tts-1","input":"hi","voice":"alloy"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "ID3fakeaudio" {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestAudioSpeechViaOpenAICompatibleType(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("compat-audio"))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("OLLAMA_API_KEY", "x")
	c := llm.NewClients(nil)
	c.OpenAICompatible = llm.NewOpenAIClient(nil)
	c.OpenAICompatible.HTTP = upstream.Client()
	// Leave OpenAI unset so resolution must use openai_compatible by route type.
	c.OpenAI = nil

	raw := "sk-compat-audio"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-compat",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_compat", Type: "openai_compatible", BaseURL: upstream.URL + "/v1",
			APIKeyEnv: "OLLAMA_API_KEY", Name: "Compat",
			Capabilities: snapshot.DefaultCapabilities("openai_compatible"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_compat", TargetModel: "tts-1",
		}},
	}))

	reg := RegistryFromClients(c)
	p := NewPipelineWithRegistry(holder, reg, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", bytes.NewBufferString(
		`{"model":"tts-1","input":"hi","voice":"alloy"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "compat-audio" {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestAudioTranscriptionsRejectsTTSModel(t *testing.T) {
	raw := "sk-stt-wrong"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: "https://api.openai.com/v1",
			APIKeyEnv: "OPENAI_API_KEY", Capabilities: snapshot.DefaultCapabilities("openai"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_openai", TargetModel: "tts-1",
		}},
	}))
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("model", "tts-1")
	part, _ := mw.CreateFormFile("file", "a.wav")
	_, _ = part.Write([]byte("RIFF"))
	_ = mw.Close()
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", &body)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAudioSpeechRejectsAnthropic(t *testing.T) {
	raw := "sk-audio-anth"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_a", Type: "anthropic", BaseURL: "https://api.anthropic.com/v1",
			APIKeyEnv: "ANTHROPIC_API_KEY", Capabilities: snapshot.DefaultCapabilities("anthropic"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_a", TargetModel: "tts-1",
		}},
	}))
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", bytes.NewBufferString(
		`{"model":"tts-1","input":"hi","voice":"alloy"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAudioTranscriptionsPassThrough(t *testing.T) {
	var sawModel string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		sawModel = r.FormValue("model")
		writeJSON(w, http.StatusOK, map[string]string{"text": "hello world"})
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("OPENAI_API_KEY", "sk-test")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()

	raw := "sk-stt-test"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: upstream.URL + "/v1",
			APIKeyEnv: "OPENAI_API_KEY", Capabilities: snapshot.DefaultCapabilities("openai"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "whisper-1", ProviderID: "prov_openai", TargetModel: "whisper-1",
		}},
	}))

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("model", "whisper-1")
	part, _ := mw.CreateFormFile("file", "a.wav")
	_, _ = part.Write([]byte("RIFF"))
	_ = mw.Close()

	p := NewPipelineWithRegistry(holder, RegistryWithOpenAI(client), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", &body)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if sawModel != "whisper-1" {
		t.Fatalf("upstream model=%q", sawModel)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("hello world")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}
