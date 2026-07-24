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

	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
)

func irToGemini(req ir.ChatRequest) ([]byte, error) {
	var contents []map[string]any
	for _, m := range req.Messages {
		switch m.Role {
		case "user", "assistant":
			role := "user"
			if m.Role == "assistant" {
				role = "model"
			}
			contents = append(contents, map[string]any{
				"role":  role,
				"parts": []map[string]string{{"text": m.Content}},
			})
		}
	}
	if len(contents) == 0 {
		return nil, fmt.Errorf("at least one user/assistant message is required")
	}
	out := map[string]any{"contents": contents}
	if req.System != "" {
		out["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": req.System}},
		}
	}
	if req.Temperature != nil {
		out["generationConfig"] = map[string]any{"temperature": *req.Temperature}
	}
	return json.Marshal(out)
}

func (c *GeminiClient) generateContentIR(ctx context.Context, provider snapshot.Provider, targetModel string, gemBody []byte, stream bool) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
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
	applyExtraHeaders(ctx, req)
	return c.HTTP.Do(req)
}

func geminiJSONToIR(raw []byte, model string) (ir.ChatResponse, error) {
	var in struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
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
		return ir.ChatResponse{}, fmt.Errorf("invalid gemini response: %w", err)
	}
	if in.Error != nil {
		return ir.ChatResponse{}, fmt.Errorf("gemini: %s", in.Error.Message)
	}
	if len(in.Candidates) == 0 {
		return ir.ChatResponse{}, fmt.Errorf("gemini: empty candidates")
	}
	var text strings.Builder
	for _, part := range in.Candidates[0].Content.Parts {
		text.WriteString(part.Text)
	}
	return ir.ChatResponse{
		ID:           "chatcmpl-gemini",
		Model:        model,
		Role:         "assistant",
		Content:      text.String(),
		FinishReason: mapGeminiFinish(in.Candidates[0].FinishReason),
		Usage: ir.Usage{
			PromptTokens:     in.UsageMetadata.PromptTokenCount,
			CompletionTokens: in.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}

func parseGeminiSSEToIR(r io.Reader, model string) <-chan ir.StreamEvent {
	ch := make(chan ir.StreamEvent, 16)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		started := false
		id := "chatcmpl-gemini"
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
				ch <- ir.StreamEvent{Kind: ir.StreamMessageStart, ID: id, Model: model, Role: "assistant"}
			}
			if text != "" {
				ch <- ir.StreamEvent{Kind: ir.StreamTextDelta, ID: id, Model: model, Text: text}
			}
			if finish != "" {
				ch <- ir.StreamEvent{Kind: ir.StreamMessageEnd, ID: id, Model: model, FinishReason: finish}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- ir.StreamEvent{Kind: ir.StreamError, Err: err}
		}
	}()
	return ch
}
