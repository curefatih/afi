package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/dataplane/openaichat"
	"github.com/curefatih/afi/internal/snapshot"
)

type GeminiClient struct {
	HTTP    *http.Client
	Secrets secrets.Resolver
}

func NewGeminiClient(sec secrets.Resolver) *GeminiClient {
	if sec == nil {
		sec = secrets.Default()
	}
	return &GeminiClient{
		HTTP:    &http.Client{Timeout: 120 * time.Second},
		Secrets: sec,
	}
}

func (c *GeminiClient) apiKey(ctx context.Context, provider snapshot.Provider) (string, error) {
	key, err := c.Secrets.Get(ctx, provider.APIKeyEnv)
	if err != nil {
		return "", fmt.Errorf("%w for provider %s", err, provider.ID)
	}
	return key, nil
}

// GenerateContent translates OpenAI chat → Gemini generateContent / streamGenerateContent.
func (c *GeminiClient) GenerateContent(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}

	gemBody, err := openAIChatToGemini(body)
	if err != nil {
		return nil, err
	}

	base := strings.TrimRight(provider.BaseURL, "/")
	method := "generateContent"
	if stream {
		method = "streamGenerateContent"
	}
	path := fmt.Sprintf("%s/models/%s:%s", base, url.PathEscape(targetModel), method)
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("key", apiKey)
	if stream {
		q.Set("alt", "sse")
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(gemBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return resp, nil
	}

	if stream {
		pr, pw := io.Pipe()
		go func() {
			defer resp.Body.Close()
			err := translateGeminiSSE(resp.Body, pw, targetModel)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type":  []string{"text/event-stream"},
				"Cache-Control": []string{"no-cache"},
			},
			Body: pr,
		}, nil
	}

	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read gemini response: %w", err)
	}

	mapped, err := geminiToOpenAIChat(raw, targetModel)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(mapped)),
	}, nil
}

func openAIChatToGemini(body []byte) ([]byte, error) {
	var in struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
		Temperature any `json:"temperature"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	var systemParts []string
	var contents []map[string]any
	for _, m := range in.Messages {
		text := openaichat.ContentToString(m.Content)
		switch m.Role {
		case "system":
			if text != "" {
				systemParts = append(systemParts, text)
			}
		case "user", "assistant":
			role := "user"
			if m.Role == "assistant" {
				role = "model"
			}
			contents = append(contents, map[string]any{
				"role": role,
				"parts": []map[string]string{
					{"text": text},
				},
			})
		}
	}
	if len(contents) == 0 {
		return nil, fmt.Errorf("at least one user/assistant message is required")
	}

	out := map[string]any{"contents": contents}
	if len(systemParts) > 0 {
		out["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": strings.Join(systemParts, "\n\n")}},
		}
	}
	if in.Temperature != nil {
		out["generationConfig"] = map[string]any{"temperature": in.Temperature}
	}
	return json.Marshal(out)
}

func geminiToOpenAIChat(raw []byte, model string) ([]byte, error) {
	var in struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int64 `json:"promptTokenCount"`
			CandidatesTokenCount int64 `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("invalid gemini response: %w", err)
	}
	if in.Error != nil {
		return nil, fmt.Errorf("gemini: %s", in.Error.Message)
	}
	if len(in.Candidates) == 0 {
		return nil, fmt.Errorf("gemini: empty candidates")
	}

	var text strings.Builder
	for _, part := range in.Candidates[0].Content.Parts {
		text.WriteString(part.Text)
	}

	finish := mapGeminiFinish(in.Candidates[0].FinishReason)

	out := map[string]any{
		"id":     "chatcmpl-gemini",
		"object": "chat.completion",
		"model":  model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": text.String(),
				},
				"finish_reason": finish,
			},
		},
		"usage": map[string]int64{
			"prompt_tokens":     in.UsageMetadata.PromptTokenCount,
			"completion_tokens": in.UsageMetadata.CandidatesTokenCount,
			"total_tokens":      in.UsageMetadata.PromptTokenCount + in.UsageMetadata.CandidatesTokenCount,
		},
	}
	return json.Marshal(out)
}

func mapGeminiFinish(r string) string {
	switch r {
	case "MAX_TOKENS":
		return "length"
	case "STOP", "":
		return "stop"
	default:
		return strings.ToLower(r)
	}
}

func translateGeminiSSE(r io.Reader, w io.Writer, model string) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	msgID := "chatcmpl-gemini"
	started := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			continue
		}

		text := extractGeminiDeltaText(raw)
		finish := extractGeminiFinish(raw)

		if !started {
			started = true
			if err := openaichat.WriteSSEChunk(w, msgID, model, map[string]any{"role": "assistant"}, nil); err != nil {
				return err
			}
		}
		if text != "" {
			if err := openaichat.WriteSSEChunk(w, msgID, model, map[string]any{"content": text}, nil); err != nil {
				return err
			}
		}
		if finish != "" {
			if err := openaichat.WriteSSEChunk(w, msgID, model, map[string]any{}, finish); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return openaichat.WriteSSEDone(w)
}

func extractGeminiDeltaText(raw map[string]any) string {
	cands, _ := raw["candidates"].([]any)
	if len(cands) == 0 {
		return ""
	}
	cand, _ := cands[0].(map[string]any)
	content, _ := cand["content"].(map[string]any)
	parts, _ := content["parts"].([]any)
	var b strings.Builder
	for _, p := range parts {
		part, _ := p.(map[string]any)
		if t, ok := part["text"].(string); ok {
			b.WriteString(t)
		}
	}
	return b.String()
}

func extractGeminiFinish(raw map[string]any) string {
	cands, _ := raw["candidates"].([]any)
	if len(cands) == 0 {
		return ""
	}
	cand, _ := cands[0].(map[string]any)
	fr, _ := cand["finishReason"].(string)
	if fr == "" {
		return ""
	}
	return mapGeminiFinish(fr)
}
