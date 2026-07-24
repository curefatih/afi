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
	started := false

	// Content block bookkeeping: Anthropic uses sequential integer indices and
	// requires the currently-open block to be closed before the next opens.
	nextIndex := 0
	curKind := "" // "" | "text" | "tool"
	curToolIR := -1
	curAnthIndex := -1
	toolAnthIndex := map[int]int{}

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

	ensureStarted := func(role string) error {
		if started {
			return nil
		}
		started = true
		if role == "" {
			role = "assistant"
		}
		return writeEvent("message_start", map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id": id, "type": "message", "role": role, "model": model, "content": []any{},
			},
		})
	}

	closeCur := func() error {
		if curKind == "" {
			return nil
		}
		idx := curAnthIndex
		curKind, curToolIR, curAnthIndex = "", -1, -1
		return writeEvent("content_block_stop", map[string]any{"type": "content_block_stop", "index": idx})
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
			if err := ensureStarted(ev.Role); err != nil {
				return prompt, completion, err
			}
		case ir.StreamTextDelta:
			if err := ensureStarted(""); err != nil {
				return prompt, completion, err
			}
			if curKind != "text" {
				if err := closeCur(); err != nil {
					return prompt, completion, err
				}
				curKind = "text"
				curAnthIndex = nextIndex
				nextIndex++
				if err := writeEvent("content_block_start", map[string]any{
					"type":          "content_block_start",
					"index":         curAnthIndex,
					"content_block": map[string]any{"type": "text", "text": ""},
				}); err != nil {
					return prompt, completion, err
				}
			}
			if ev.Text == "" {
				continue
			}
			if err := writeEvent("content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": curAnthIndex,
				"delta": map[string]any{"type": "text_delta", "text": ev.Text},
			}); err != nil {
				return prompt, completion, err
			}
		case ir.StreamToolCallStart:
			if err := ensureStarted(""); err != nil {
				return prompt, completion, err
			}
			if err := closeCur(); err != nil {
				return prompt, completion, err
			}
			curKind = "tool"
			curToolIR = ev.ToolIndex
			curAnthIndex = nextIndex
			nextIndex++
			toolAnthIndex[ev.ToolIndex] = curAnthIndex
			if err := writeEvent("content_block_start", map[string]any{
				"type":  "content_block_start",
				"index": curAnthIndex,
				"content_block": map[string]any{
					"type": "tool_use", "id": ev.ToolID, "name": ev.ToolName, "input": map[string]any{},
				},
			}); err != nil {
				return prompt, completion, err
			}
		case ir.StreamToolCallDelta:
			if ev.ArgsDelta == "" {
				continue
			}
			if curKind != "tool" || curToolIR != ev.ToolIndex {
				// Deltas are expected to be contiguous with their tool_use block.
				continue
			}
			if err := writeEvent("content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": curAnthIndex,
				"delta": map[string]any{"type": "input_json_delta", "partial_json": ev.ArgsDelta},
			}); err != nil {
				return prompt, completion, err
			}
		case ir.StreamMessageEnd:
			if err := closeCur(); err != nil {
				return prompt, completion, err
			}
			finish := ev.FinishReason
			if finish == "" {
				finish = "stop"
			}
			delta := map[string]any{"stop_reason": mapFinishToAnthropic(finish)}
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

func anthropicBlockIndex(raw map[string]any) int {
	if v, ok := raw["index"].(float64); ok {
		return int(v)
	}
	return 0
}

func mapFinishToAnthropic(r string) string {
	switch r {
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
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
		toolOrdinal := map[int]int{}
		nextToolOrdinal := 0
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
				blockIndex := anthropicBlockIndex(raw)
				if block, ok := raw["content_block"].(map[string]any); ok {
					switch typ, _ := block["type"].(string); typ {
					case "text":
						if text, ok := block["text"].(string); ok && text != "" {
							ch <- ir.StreamEvent{Kind: ir.StreamTextDelta, ID: id, Model: model, Text: text}
						}
					case "tool_use":
						ordinal := nextToolOrdinal
						nextToolOrdinal++
						toolOrdinal[blockIndex] = ordinal
						toolID, _ := block["id"].(string)
						toolName, _ := block["name"].(string)
						ch <- ir.StreamEvent{Kind: ir.StreamToolCallStart, ID: id, Model: model, ToolIndex: ordinal, ToolID: toolID, ToolName: toolName}
					}
				}
			case "content_block_delta":
				blockIndex := anthropicBlockIndex(raw)
				if delta, ok := raw["delta"].(map[string]any); ok {
					dtyp, _ := delta["type"].(string)
					switch dtyp {
					case "input_json_delta":
						if pj, ok := delta["partial_json"].(string); ok && pj != "" {
							ch <- ir.StreamEvent{Kind: ir.StreamToolCallDelta, ID: id, Model: model, ToolIndex: toolOrdinal[blockIndex], ArgsDelta: pj}
						}
					default:
						if text, ok := delta["text"].(string); ok && text != "" {
							ch <- ir.StreamEvent{Kind: ir.StreamTextDelta, ID: id, Model: model, Text: text}
						}
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
