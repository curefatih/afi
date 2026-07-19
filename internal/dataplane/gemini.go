package dataplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/snapshot"
)

type GeminiClient struct {
	HTTP *http.Client
}

func NewGeminiClient() *GeminiClient {
	return &GeminiClient{
		HTTP: &http.Client{Timeout: 120 * time.Second},
	}
}

// GenerateContent translates OpenAI chat → Gemini generateContent and maps back.
func (c *GeminiClient) GenerateContent(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error) {
	apiKey := os.Getenv(provider.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("missing env %s for provider %s", provider.APIKeyEnv, provider.ID)
	}

	gemBody, err := openAIChatToGemini(body)
	if err != nil {
		return nil, err
	}

	base := strings.TrimRight(provider.BaseURL, "/")
	path := fmt.Sprintf("%s/models/%s:generateContent", base, url.PathEscape(targetModel))
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("key", apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(gemBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return resp, nil
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
		text := contentToString(m.Content)
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

	finish := "stop"
	switch in.Candidates[0].FinishReason {
	case "MAX_TOKENS":
		finish = "length"
	case "STOP", "":
		finish = "stop"
	default:
		finish = strings.ToLower(in.Candidates[0].FinishReason)
	}

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
