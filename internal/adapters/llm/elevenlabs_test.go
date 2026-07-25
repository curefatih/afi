package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestElevenLabsAudioSpeech(t *testing.T) {
	var gotPath, gotKey, gotAccept string
	var gotBody map[string]any
	var gotQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotKey = r.Header.Get("xi-api-key")
		gotAccept = r.Header.Get("Accept")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3eleven"))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("ELEVENLABS_API_KEY", "el-test-key")
	client := NewElevenLabsClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_el", Type: "elevenlabs", BaseURL: upstream.URL, APIKeyEnv: "ELEVENLABS_API_KEY",
	}
	body := []byte(`{"model":"eleven_multilingual_v2","input":"hello","voice":"alloy","response_format":"mp3"}`)
	resp, err := client.AudioSpeech(t.Context(), provider, "eleven_multilingual_v2", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if gotKey != "el-test-key" {
		t.Fatalf("xi-api-key=%q", gotKey)
	}
	if gotAccept != "audio/mpeg" {
		t.Fatalf("accept=%q", gotAccept)
	}
	if gotPath != "/v1/text-to-speech/"+defaultElevenLabsVoiceID {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.Contains(gotQuery, "output_format=mp3_44100_128") {
		t.Fatalf("query=%q", gotQuery)
	}
	if gotBody["text"] != "hello" || gotBody["model_id"] != "eleven_multilingual_v2" {
		t.Fatalf("body=%v", gotBody)
	}
	raw, _ := io.ReadAll(resp.Body)
	if string(raw) != "ID3eleven" {
		t.Fatalf("audio=%q", raw)
	}
}

func TestElevenLabsAudioSpeechPassesVoiceID(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	client := NewElevenLabsClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_el", Type: "elevenlabs", BaseURL: upstream.URL, InlineAPIKey: "k",
	}
	voiceID := "JBFqnCBsd6RMkjVDRZzb"
	body := []byte(`{"input":"hi","voice":"` + voiceID + `"}`)
	resp, err := client.AudioSpeech(t.Context(), provider, "eleven_flash_v2_5", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if gotPath != "/v1/text-to-speech/"+voiceID {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestElevenLabsAudioTranscriptions(t *testing.T) {
	var gotPath, gotKey string
	var gotModelID, gotLang string
	var gotFile []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("xi-api-key")
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatalf("parse: %v", err)
		}
		gotModelID = r.FormValue("model_id")
		gotLang = r.FormValue("language_code")
		if r.FormValue("model") != "" {
			t.Fatal("openai model field should be renamed")
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("file: %v", err)
		}
		defer f.Close()
		gotFile, _ = io.ReadAll(f)
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "transcribed"})
	}))
	t.Cleanup(upstream.Close)

	client := NewElevenLabsClient(nil)
	client.HTTP = upstream.Client()
	provider := snapshot.Provider{
		ID: "prov_el", Type: "elevenlabs", BaseURL: upstream.URL, InlineAPIKey: "el-key",
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("model", "whisper-1")
	_ = mw.WriteField("language", "en")
	_ = mw.WriteField("prompt", "ignore me")
	part, _ := mw.CreateFormFile("file", "a.wav")
	_, _ = part.Write([]byte("RIFFDATA"))
	_ = mw.Close()

	resp, err := client.AudioTranscriptions(t.Context(), provider, "scribe_v2", mw.FormDataContentType(), &body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if gotPath != "/v1/speech-to-text" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotKey != "el-key" {
		t.Fatalf("key=%q", gotKey)
	}
	if gotModelID != "scribe_v2" {
		t.Fatalf("model_id=%q", gotModelID)
	}
	if gotLang != "en" {
		t.Fatalf("language_code=%q", gotLang)
	}
	if string(gotFile) != "RIFFDATA" {
		t.Fatalf("file=%q", gotFile)
	}
}

func TestResolveElevenLabsVoiceID(t *testing.T) {
	t.Parallel()
	if got := resolveElevenLabsVoiceID(""); got != defaultElevenLabsVoiceID {
		t.Fatalf("empty=%q", got)
	}
	if got := resolveElevenLabsVoiceID("nova"); got != defaultElevenLabsVoiceID {
		t.Fatalf("nova=%q", got)
	}
	if got := resolveElevenLabsVoiceID("customVoice123"); got != "customVoice123" {
		t.Fatalf("custom=%q", got)
	}
}

func TestElevenLabsOutputFormat(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":               "mp3_44100_128",
		"mp3":            "mp3_44100_128",
		"opus":           "opus_48000_128",
		"pcm":            "pcm_16000",
		"wav":            "pcm_44100",
		"mp3_22050_32":   "mp3_22050_32",
		"unknown-format": "mp3_44100_128",
	}
	for in, want := range cases {
		if got := elevenLabsOutputFormat(in); got != want {
			t.Fatalf("%q → %q want %q", in, got, want)
		}
	}
}
