package dialect

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/dataplane/openaichat"
)

// OpenAI is the OpenAI chat.completions dialect codec.
type OpenAI struct{}

func (OpenAI) Name() ir.Dialect { return ir.DialectOpenAI }

func (OpenAI) DecodeRequest(body []byte) (ir.ChatRequest, error) {
	return ir.DecodeOpenAIRequest(body)
}

func (OpenAI) EncodeResponse(resp ir.ChatResponse) ([]byte, error) {
	return ir.EncodeOpenAIResponse(resp)
}

func (OpenAI) WriteStream(w io.Writer, events <-chan ir.StreamEvent) (prompt, completion int64, err error) {
	flusher, _ := w.(http.Flusher)
	id := "chatcmpl-afi"
	model := ""
	for ev := range events {
		switch ev.Kind {
		case ir.StreamError:
			if ev.Err != nil {
				return prompt, completion, ev.Err
			}
			return prompt, completion, fmt.Errorf("stream error")
		case ir.StreamMessageStart:
			if ev.ID != "" {
				id = ev.ID
			}
			if ev.Model != "" {
				model = ev.Model
			}
			role := ev.Role
			if role == "" {
				role = "assistant"
			}
			if err := openaichat.WriteSSEChunk(w, id, model, map[string]any{"role": role}, nil); err != nil {
				return prompt, completion, err
			}
		case ir.StreamTextDelta:
			if ev.Text == "" {
				continue
			}
			if err := openaichat.WriteSSEChunk(w, id, model, map[string]any{"content": ev.Text}, nil); err != nil {
				return prompt, completion, err
			}
		case ir.StreamMessageEnd:
			var finish any
			if ev.FinishReason != "" {
				finish = ev.FinishReason
			}
			if err := openaichat.WriteSSEChunk(w, id, model, map[string]any{}, finish); err != nil {
				return prompt, completion, err
			}
			if ev.Usage != nil {
				prompt, completion = ev.Usage.PromptTokens, ev.Usage.CompletionTokens
				usageChunk := map[string]any{
					"id":      id,
					"object":  "chat.completion.chunk",
					"model":   model,
					"choices": []map[string]any{},
					"usage": map[string]int64{
						"prompt_tokens":     prompt,
						"completion_tokens": completion,
						"total_tokens":      prompt + completion,
					},
				}
				b, mErr := json.Marshal(usageChunk)
				if mErr != nil {
					return prompt, completion, mErr
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
					return prompt, completion, err
				}
			}
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	return prompt, completion, openaichat.WriteSSEDone(w)
}

// ParseOpenAISSE reads OpenAI chat.completion.chunk SSE into IR stream events.
func ParseOpenAISSE(r io.Reader) <-chan ir.StreamEvent {
	ch := make(chan ir.StreamEvent, 16)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		started := false
		id := "chatcmpl-afi"
		model := ""
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}
			var raw struct {
				ID      string `json:"id"`
				Model   string `json:"model"`
				Choices []struct {
					Delta struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason *string `json:"finish_reason"`
				} `json:"choices"`
				Usage *struct {
					PromptTokens     int64 `json:"prompt_tokens"`
					CompletionTokens int64 `json:"completion_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(payload), &raw); err != nil {
				continue
			}
			if raw.ID != "" {
				id = raw.ID
			}
			if raw.Model != "" {
				model = raw.Model
			}
			if raw.Usage != nil && (raw.Usage.PromptTokens > 0 || raw.Usage.CompletionTokens > 0) {
				u := &ir.Usage{PromptTokens: raw.Usage.PromptTokens, CompletionTokens: raw.Usage.CompletionTokens}
				if len(raw.Choices) == 0 {
					ch <- ir.StreamEvent{Kind: ir.StreamMessageEnd, ID: id, Model: model, Usage: u}
					continue
				}
			}
			if len(raw.Choices) == 0 {
				continue
			}
			c := raw.Choices[0]
			if !started && (c.Delta.Role != "" || c.Delta.Content != "" || c.FinishReason != nil) {
				started = true
				role := c.Delta.Role
				if role == "" {
					role = "assistant"
				}
				ch <- ir.StreamEvent{Kind: ir.StreamMessageStart, ID: id, Model: model, Role: role}
			}
			if c.Delta.Content != "" {
				ch <- ir.StreamEvent{Kind: ir.StreamTextDelta, ID: id, Model: model, Text: c.Delta.Content}
			}
			if c.FinishReason != nil && *c.FinishReason != "" {
				ev := ir.StreamEvent{Kind: ir.StreamMessageEnd, ID: id, Model: model, FinishReason: *c.FinishReason}
				if raw.Usage != nil {
					ev.Usage = &ir.Usage{PromptTokens: raw.Usage.PromptTokens, CompletionTokens: raw.Usage.CompletionTokens}
				}
				ch <- ev
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- ir.StreamEvent{Kind: ir.StreamError, Err: err}
		}
	}()
	return ch
}
