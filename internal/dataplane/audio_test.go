package dataplane

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
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

func TestAudioSpeechElevenLabs(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("xi-api-key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("el-audio"))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("ELEVENLABS_API_KEY", "el-gateway")
	c := llm.NewClients(nil)
	c.ElevenLabs = llm.NewElevenLabsClient(nil)
	c.ElevenLabs.HTTP = upstream.Client()

	raw := "sk-eleven-audio"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-eleven",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_el", Type: "elevenlabs", BaseURL: upstream.URL,
			APIKeyEnv: "ELEVENLABS_API_KEY", Name: "ElevenLabs",
			Capabilities: snapshot.DefaultCapabilities("elevenlabs"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "eleven-tts", ProviderID: "prov_el", TargetModel: "eleven_multilingual_v2",
		}},
	}))

	p := NewPipelineWithRegistry(holder, RegistryFromClients(c), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", bytes.NewBufferString(
		`{"model":"eleven-tts","input":"hello eleven","voice":"alloy"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "el-audio" {
		t.Fatalf("body=%q", rr.Body.String())
	}
	if gotKey != "el-gateway" {
		t.Fatalf("xi-api-key=%q", gotKey)
	}
	if !strings.HasPrefix(gotPath, "/v1/text-to-speech/") {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["text"] != "hello eleven" || gotBody["model_id"] != "eleven_multilingual_v2" {
		t.Fatalf("body=%v", gotBody)
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

func TestAudioSpeechFailoverPrimary500UsesFallback(t *testing.T) {
	var primaryHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("from-fallback"))
	}))
	t.Cleanup(fallback.Close)

	t.Setenv("OPENAI_API_KEY", "k")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()

	raw := "sk-audio-fo"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_a", TargetModel: "tts-1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "tts-1"}},
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
	if rr.Body.String() != "from-fallback" {
		t.Fatalf("body=%q", rr.Body.String())
	}
	if primaryHits.Load() != 1 {
		t.Fatalf("primary hits=%d", primaryHits.Load())
	}
}

func TestAudioSpeechRetrySameTargetBeforeFallback(t *testing.T) {
	var primaryHits, fallbackHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := primaryHits.Add(1)
		if n < 3 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("after-retry"))
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits.Add(1)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("should-not"))
	}))
	t.Cleanup(fallback.Close)

	t.Setenv("OPENAI_API_KEY", "k")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()

	raw := "sk-audio-retry"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_a", TargetModel: "tts-1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "tts-1"}},
			Retry: &snapshot.RetryConfig{
				MaxAttempts: 3,
				Backoff:     snapshot.BackoffConfig{Strategy: snapshot.BackoffFixed, BaseDelay: "1ms"},
			},
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
	if rr.Body.String() != "after-retry" {
		t.Fatalf("body=%q", rr.Body.String())
	}
	if primaryHits.Load() != 3 {
		t.Fatalf("primary hits=%d", primaryHits.Load())
	}
	if fallbackHits.Load() != 0 {
		t.Fatalf("fallback hits=%d", fallbackHits.Load())
	}
}

func TestAudioTranscriptionsFailover429(t *testing.T) {
	var primaryHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		http.Error(w, "rate", http.StatusTooManyRequests)
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		writeJSON(w, http.StatusOK, map[string]string{"text": "from-fallback"})
	}))
	t.Cleanup(fallback.Close)

	t.Setenv("OPENAI_API_KEY", "k")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()

	raw := "sk-stt-fo"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "whisper-1", ProviderID: "prov_a", TargetModel: "whisper-1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "whisper-1"}},
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
	if !bytes.Contains(rr.Body.Bytes(), []byte("from-fallback")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if primaryHits.Load() != 1 {
		t.Fatalf("primary hits=%d", primaryHits.Load())
	}
}

func TestAudioSpeechSkipsFallbackWithoutTTS(t *testing.T) {
	var primaryHits, noTTSHits, goodHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(primary.Close)

	noTTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noTTSHits.Add(1)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("no-tts"))
	}))
	t.Cleanup(noTTS.Close)

	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		goodHits.Add(1)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("good-tts"))
	}))
	t.Cleanup(good.Close)

	t.Setenv("OPENAI_API_KEY", "k")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()

	raw := "sk-audio-skip"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
			// Chat+STT set so NormalizeCapabilities does not promote TTS from defaults.
			{ID: "prov_notts", Type: "openai", BaseURL: noTTS.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.ProviderCapabilities{Chat: true, Stream: true, TTS: false, STT: true}},
			{ID: "prov_b", Type: "openai", BaseURL: good.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_a", TargetModel: "tts-1",
			Fallbacks: []snapshot.RouteTarget{
				{ProviderID: "prov_notts", TargetModel: "tts-1"},
				{ProviderID: "prov_b", TargetModel: "tts-1"},
			},
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
	if rr.Body.String() != "good-tts" {
		t.Fatalf("body=%q", rr.Body.String())
	}
	if noTTSHits.Load() != 0 {
		t.Fatalf("no-tts hits=%d", noTTSHits.Load())
	}
	if goodHits.Load() != 1 {
		t.Fatalf("good hits=%d", goodHits.Load())
	}
}

func TestAudioSpeechNonFailover4xxDoesNotWalkFallbacks(t *testing.T) {
	var primaryHits, fallbackHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits.Add(1)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("should-not"))
	}))
	t.Cleanup(fallback.Close)

	t.Setenv("OPENAI_API_KEY", "k")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()

	raw := "sk-audio-4xx"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "o1", ProjectID: "p1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL + "/v1", APIKeyEnv: "OPENAI_API_KEY",
				Capabilities: snapshot.DefaultCapabilities("openai")},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "tts-1", ProviderID: "prov_a", TargetModel: "tts-1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "tts-1"}},
		}},
	}))

	p := NewPipelineWithRegistry(holder, RegistryWithOpenAI(client), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", bytes.NewBufferString(
		`{"model":"tts-1","input":"hi","voice":"alloy"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if primaryHits.Load() != 1 {
		t.Fatalf("primary hits=%d", primaryHits.Load())
	}
	if fallbackHits.Load() != 0 {
		t.Fatalf("fallback hits=%d", fallbackHits.Load())
	}
}
