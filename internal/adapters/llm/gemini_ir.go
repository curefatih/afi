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

	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
)

func irToGemini(req ir.ChatRequest) ([]byte, error) {
	return dialect.EncodeGeminiRequest(req)
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
					Text         string `json:"text"`
					FunctionCall *struct {
						ID   string         `json:"id"`
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
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
	var toolCalls []ir.ToolCall
	callNumber := 0
	for _, part := range in.Candidates[0].Content.Parts {
		text.WriteString(part.Text)
		if part.FunctionCall != nil {
			id := part.FunctionCall.ID
			if id == "" {
				id = fmt.Sprintf("call_gemini_%d", callNumber)
			}
			callNumber++
			args, _ := json.Marshal(part.FunctionCall.Args)
			toolCalls = append(toolCalls, ir.ToolCall{
				ID: id, Name: part.FunctionCall.Name, Arguments: string(args),
			})
		}
	}
	finishReason := mapGeminiFinish(in.Candidates[0].FinishReason)
	if len(toolCalls) > 0 && finishReason == "stop" {
		finishReason = "tool_calls"
	}
	return ir.ChatResponse{
		ID:           "chatcmpl-gemini",
		Model:        model,
		Role:         "assistant",
		Content:      text.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
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
		nextToolIndex := 0
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
			calls := extractGeminiFunctionCalls(raw)
			finish := extractGeminiFinish(raw)
			if !started {
				started = true
				ch <- ir.StreamEvent{Kind: ir.StreamMessageStart, ID: id, Model: model, Role: "assistant"}
			}
			if text != "" {
				ch <- ir.StreamEvent{Kind: ir.StreamTextDelta, ID: id, Model: model, Text: text}
			}
			for _, call := range calls {
				ch <- ir.StreamEvent{
					Kind: ir.StreamToolCallStart, ID: id, Model: model,
					ToolIndex: nextToolIndex, ToolID: call.ID, ToolName: call.Name,
				}
				ch <- ir.StreamEvent{
					Kind: ir.StreamToolCallDelta, ID: id, Model: model,
					ToolIndex: nextToolIndex, ArgsDelta: call.Arguments,
				}
				nextToolIndex++
			}
			if finish != "" {
				if nextToolIndex > 0 && finish == "stop" {
					finish = "tool_calls"
				}
				ch <- ir.StreamEvent{Kind: ir.StreamMessageEnd, ID: id, Model: model, FinishReason: finish}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- ir.StreamEvent{Kind: ir.StreamError, Err: err}
		}
	}()
	return ch
}

func extractGeminiFunctionCalls(raw map[string]any) []ir.ToolCall {
	candidates, _ := raw["candidates"].([]any)
	if len(candidates) == 0 {
		return nil
	}
	candidate, _ := candidates[0].(map[string]any)
	content, _ := candidate["content"].(map[string]any)
	parts, _ := content["parts"].([]any)
	var calls []ir.ToolCall
	for i, partRaw := range parts {
		part, _ := partRaw.(map[string]any)
		functionCall, _ := part["functionCall"].(map[string]any)
		if functionCall == nil {
			continue
		}
		id, _ := functionCall["id"].(string)
		if id == "" {
			id = fmt.Sprintf("call_gemini_%d", i)
		}
		name, _ := functionCall["name"].(string)
		args, _ := json.Marshal(functionCall["args"])
		calls = append(calls, ir.ToolCall{ID: id, Name: name, Arguments: string(args)})
	}
	return calls
}
