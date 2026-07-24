package dialect

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

// Anthropic is the Anthropic messages dialect codec.
type Anthropic struct{}

func (Anthropic) Name() ir.Dialect { return ir.DialectAnthropic }

func (Anthropic) DecodeRequest(body []byte) (ir.ChatRequest, error) {
	return ir.DecodeAnthropicRequest(body)
}

func (Anthropic) EncodeResponse(resp ir.ChatResponse) ([]byte, error) {
	return ir.EncodeAnthropicResponse(resp)
}

func (Anthropic) WriteStream(w io.Writer, events <-chan ir.StreamEvent) (prompt, completion int64, err error) {
	flusher, _ := w.(http.Flusher)
	id := "msg_afi"
	model := ""
	index := 0
	started := false
	blockStarted := false

	writeEvent := func(typ string, payload any) error {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", typ, b)
		if flusher != nil {
			flusher.Flush()
		}
		return err
	}

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
			started = true
			if err := writeEvent("message_start", map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id":      id,
					"type":    "message",
					"role":    role,
					"model":   model,
					"content": []any{},
				},
			}); err != nil {
				return prompt, completion, err
			}
		case ir.StreamTextDelta:
			if !started {
				started = true
				if err := writeEvent("message_start", map[string]any{
					"type": "message_start",
					"message": map[string]any{
						"id": id, "type": "message", "role": "assistant", "model": model, "content": []any{},
					},
				}); err != nil {
					return prompt, completion, err
				}
			}
			if !blockStarted {
				blockStarted = true
				if err := writeEvent("content_block_start", map[string]any{
					"type":  "content_block_start",
					"index": index,
					"content_block": map[string]any{
						"type": "text",
						"text": "",
					},
				}); err != nil {
					return prompt, completion, err
				}
			}
			if ev.Text == "" {
				continue
			}
			if err := writeEvent("content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": index,
				"delta": map[string]any{"type": "text_delta", "text": ev.Text},
			}); err != nil {
				return prompt, completion, err
			}
		case ir.StreamMessageEnd:
			if blockStarted {
				if err := writeEvent("content_block_stop", map[string]any{
					"type": "content_block_stop", "index": index,
				}); err != nil {
					return prompt, completion, err
				}
			}
			finish := ev.FinishReason
			if finish == "" {
				finish = "stop"
			}
			delta := map[string]any{
				"stop_reason": mapFinishToAnthropic(finish),
			}
			usage := map[string]any{}
			if ev.Usage != nil {
				prompt, completion = ev.Usage.PromptTokens, ev.Usage.CompletionTokens
				usage["input_tokens"] = prompt
				usage["output_tokens"] = completion
			}
			payload := map[string]any{"type": "message_delta", "delta": delta}
			if len(usage) > 0 {
				payload["usage"] = usage
			}
			if err := writeEvent("message_delta", payload); err != nil {
				return prompt, completion, err
			}
			if err := writeEvent("message_stop", map[string]any{"type": "message_stop"}); err != nil {
				return prompt, completion, err
			}
		}
	}
	return prompt, completion, nil
}

func mapFinishToAnthropic(r string) string {
	switch r {
	case "length":
		return "max_tokens"
	case "stop", "":
		return "end_turn"
	default:
		return r
	}
}

// ParseAnthropicSSE reads Anthropic messages SSE into IR stream events.
func ParseAnthropicSSE(r io.Reader) <-chan ir.StreamEvent {
	ch := make(chan ir.StreamEvent, 16)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		id := "msg_afi"
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
			var raw map[string]any
			if err := json.Unmarshal([]byte(payload), &raw); err != nil {
				continue
			}
			evType, _ := raw["type"].(string)
			switch evType {
			case "message_start":
				if msg, ok := raw["message"].(map[string]any); ok {
					if v, ok := msg["id"].(string); ok && v != "" {
						id = v
					}
					if v, ok := msg["model"].(string); ok && v != "" {
						model = v
					}
					role, _ := msg["role"].(string)
					if role == "" {
						role = "assistant"
					}
					ch <- ir.StreamEvent{Kind: ir.StreamMessageStart, ID: id, Model: model, Role: role}
				}
			case "content_block_start":
				if block, ok := raw["content_block"].(map[string]any); ok {
					if typ, _ := block["type"].(string); typ == "text" {
						if text, ok := block["text"].(string); ok && text != "" {
							ch <- ir.StreamEvent{Kind: ir.StreamTextDelta, ID: id, Model: model, Text: text}
						}
					}
				}
			case "content_block_delta":
				if delta, ok := raw["delta"].(map[string]any); ok {
					if text, ok := delta["text"].(string); ok && text != "" {
						ch <- ir.StreamEvent{Kind: ir.StreamTextDelta, ID: id, Model: model, Text: text}
					}
				}
			case "message_delta":
				ev := ir.StreamEvent{Kind: ir.StreamMessageEnd, ID: id, Model: model}
				if delta, ok := raw["delta"].(map[string]any); ok {
					if sr, ok := delta["stop_reason"].(string); ok {
						ev.FinishReason = ir.MapAnthropicStopReason(sr)
					}
				}
				if usage, ok := raw["usage"].(map[string]any); ok {
					u := &ir.Usage{}
					if v, ok := usage["input_tokens"].(float64); ok {
						u.PromptTokens = int64(v)
					}
					if v, ok := usage["output_tokens"].(float64); ok {
						u.CompletionTokens = int64(v)
					}
					ev.Usage = u
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
