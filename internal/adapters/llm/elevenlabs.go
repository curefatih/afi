package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/telemetry"
)

const (
	defaultElevenLabsVoiceID = "21m00Tcm4TlvDq8ikWAM" // Rachel
	defaultElevenLabsBaseURL = "https://api.elevenlabs.io"
)

// ElevenLabsClient calls ElevenLabs TTS/STT APIs, translating OpenAI audio dialect bodies.
type ElevenLabsClient struct {
	HTTP    *http.Client
	Secrets secrets.Resolver
}

// NewElevenLabsClient constructs an ElevenLabs adapter with a shared secret resolver.
func NewElevenLabsClient(sec secrets.Resolver) *ElevenLabsClient {
	if sec == nil {
		sec = secrets.Default()
	}
	return &ElevenLabsClient{
		HTTP:    &http.Client{Timeout: 120 * time.Second, Transport: telemetry.WrapTransport(http.DefaultTransport)},
		Secrets: sec,
	}
}

func (c *ElevenLabsClient) apiKey(ctx context.Context, provider snapshot.Provider) (string, error) {
	if provider.InlineAPIKey != "" {
		return provider.InlineAPIKey, nil
	}
	key, err := c.Secrets.Get(ctx, provider.APIKeyEnv)
	if err != nil {
		return "", fmt.Errorf("%w for provider %s", err, provider.ID)
	}
	return key, nil
}

func (c *ElevenLabsClient) baseURL(provider snapshot.Provider) string {
	base := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if base == "" {
		return defaultElevenLabsBaseURL
	}
	return base
}

// AudioSpeech translates OpenAI /v1/audio/speech JSON to ElevenLabs text-to-speech.
//
// OpenAI body fields: model, input, voice, response_format, speed
// ElevenLabs: POST /v1/text-to-speech/{voice_id}?output_format=… with {text, model_id}
func (c *ElevenLabsClient) AudioSpeech(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}
	var req struct {
		Model          string `json:"model"`
		Input          string `json:"input"`
		Voice          string `json:"voice"`
		ResponseFormat string `json:"response_format"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	if strings.TrimSpace(req.Input) == "" {
		return nil, fmt.Errorf("input is required")
	}
	modelID := strings.TrimSpace(targetModel)
	if modelID == "" {
		modelID = strings.TrimSpace(req.Model)
	}
	if modelID == "" {
		return nil, fmt.Errorf("model is required")
	}
	voiceID := resolveElevenLabsVoiceID(req.Voice)
	payload := map[string]any{
		"text":     req.Input,
		"model_id": modelID,
	}
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(c.baseURL(provider) + "/v1/text-to-speech/" + url.PathEscape(voiceID))
	if err != nil {
		return nil, err
	}
	if of := elevenLabsOutputFormat(req.ResponseFormat); of != "" {
		q := u.Query()
		q.Set("output_format", of)
		u.RawQuery = q.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("xi-api-key", apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "audio/mpeg")
	applyExtraHeaders(ctx, httpReq)
	return c.HTTP.Do(httpReq)
}

// AudioTranscriptions translates OpenAI multipart transcriptions to ElevenLabs speech-to-text.
//
// OpenAI fields: model, file, language, …
// ElevenLabs: POST /v1/speech-to-text with model_id + file (+ language_code).
func (c *ElevenLabsClient) AudioTranscriptions(ctx context.Context, provider snapshot.Provider, targetModel, contentType string, body io.Reader) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}
	rewritten, newCT, err := rewriteElevenLabsSTTMultipart(contentType, body, targetModel)
	if err != nil {
		return nil, err
	}
	url := c.baseURL(provider) + "/v1/speech-to-text"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("xi-api-key", apiKey)
	httpReq.Header.Set("Content-Type", newCT)
	applyExtraHeaders(ctx, httpReq)
	return c.HTTP.Do(httpReq)
}

// resolveElevenLabsVoiceID maps OpenAI-style voice names to ElevenLabs voice IDs.
// Unknown non-empty values are passed through (assumed to already be voice IDs).
func resolveElevenLabsVoiceID(voice string) string {
	v := strings.TrimSpace(voice)
	if v == "" {
		return defaultElevenLabsVoiceID
	}
	switch strings.ToLower(v) {
	case "alloy", "ash", "ballad", "coral", "echo", "fable", "onyx", "nova", "sage", "shimmer":
		return defaultElevenLabsVoiceID
	default:
		return v
	}
}

// elevenLabsOutputFormat maps OpenAI response_format to an ElevenLabs output_format query value.
func elevenLabsOutputFormat(responseFormat string) string {
	switch strings.ToLower(strings.TrimSpace(responseFormat)) {
	case "", "mp3":
		return "mp3_44100_128"
	case "opus":
		return "opus_48000_128"
	case "pcm":
		return "pcm_16000"
	case "wav":
		return "pcm_44100"
	case "ulaw", "mulaw", "mu-law", "u-law":
		return "ulaw_8000"
	default:
		// Pass through already-ElevenLabs-shaped values (e.g. mp3_44100_128).
		if strings.Contains(responseFormat, "_") {
			return responseFormat
		}
		return "mp3_44100_128"
	}
}

// rewriteElevenLabsSTTMultipart rebuilds OpenAI transcription multipart as ElevenLabs STT form.
// Renames model → model_id and language → language_code; drops OpenAI-only fields.
func rewriteElevenLabsSTTMultipart(contentType string, body io.Reader, targetModel string) ([]byte, string, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", fmt.Errorf("content-type: %w", err)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, "", fmt.Errorf("multipart boundary missing")
	}
	mr := multipart.NewReader(body, boundary)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	modelWritten := false
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
		name := part.FormName()
		switch name {
		case "model":
			w, err := mw.CreateFormField("model_id")
			if err != nil {
				return nil, "", err
			}
			if _, err := io.WriteString(w, targetModel); err != nil {
				return nil, "", err
			}
			modelWritten = true
			_, _ = io.Copy(io.Discard, part)
		case "language":
			w, err := mw.CreateFormField("language_code")
			if err != nil {
				return nil, "", err
			}
			if _, err := io.Copy(w, part); err != nil {
				return nil, "", err
			}
		case "prompt", "response_format", "temperature":
			_, _ = io.Copy(io.Discard, part)
		default:
			hdr := textproto.MIMEHeader{}
			for k, vals := range part.Header {
				for _, v := range vals {
					hdr.Add(k, v)
				}
			}
			w, err := mw.CreatePart(hdr)
			if err != nil {
				return nil, "", err
			}
			if _, err := io.Copy(w, part); err != nil {
				return nil, "", err
			}
		}
	}
	if !modelWritten {
		w, err := mw.CreateFormField("model_id")
		if err != nil {
			return nil, "", err
		}
		if _, err := io.WriteString(w, targetModel); err != nil {
			return nil, "", err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), mw.FormDataContentType(), nil
}
