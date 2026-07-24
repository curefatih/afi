package ir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EncodeAnthropic marshals a ChatRequest to Anthropic messages JSON.
func EncodeAnthropic(req ChatRequest, targetModel string) ([]byte, error) {
	model := targetModel
	if model == "" {
		model = req.Model
	}
	messages := anthropicMessages(req.Messages)
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one user/assistant message is required")
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	out := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if req.Stream {
		out["stream"] = true
	}
	sys := req.System
	if sys == "" {
		var parts []string
		for _, m := range req.Messages {
			if m.Role == "system" && m.Content != "" {
				parts = append(parts, m.Content)
			}
		}
		sys = strings.Join(parts, "\n\n")
	}
	if sys != "" {
		out["system"] = sys
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		out["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		out["stop_sequences"] = req.Stop
	}
	if len(req.Tools) > 0 {
		out["tools"] = anthropicTools(req.Tools)
	}
	if tc := anthropicToolChoice(req.ToolChoice); tc != nil {
		out["tool_choice"] = tc
	}
	return json.Marshal(out)
}

// anthropicMessages converts IR messages into Anthropic messages, coalescing
// consecutive user/tool turns into a single user message so tool_result blocks
// stay in a user turn and role alternation is preserved.
func anthropicMessages(msgs []Message) []map[string]any {
	var messages []map[string]any
	var curRole string
	var curBlocks []any

	flush := func() {
		if curRole != "" && len(curBlocks) > 0 {
			messages = append(messages, map[string]any{"role": curRole, "content": curBlocks})
		}
		curRole = ""
		curBlocks = nil
	}

	for _, m := range msgs {
		switch m.Role {
		case "system":
			continue
		case "assistant":
			if curRole != "assistant" {
				flush()
				curRole = "assistant"
			}
			curBlocks = append(curBlocks, assistantBlocks(m)...)
		case "user":
			if curRole != "user" {
				flush()
				curRole = "user"
			}
			curBlocks = append(curBlocks, contentBlocks(m)...)
		case "tool":
			if curRole != "user" {
				flush()
				curRole = "user"
			}
			curBlocks = append(curBlocks, map[string]any{
				"type":        "tool_result",
				"tool_use_id": m.ToolCallID,
				"content":     m.Content,
			})
		}
	}
	flush()
	return messages
}

func assistantBlocks(m Message) []any {
	blocks := contentBlocks(m)
	for _, tc := range m.ToolCalls {
		blocks = append(blocks, map[string]any{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": argsToObject(tc.Arguments),
		})
	}
	return blocks
}

func contentBlocks(m Message) []any {
	var blocks []any
	if len(m.Parts) > 0 {
		for _, p := range m.Parts {
			switch p.Type {
			case ContentImage:
				if b := anthropicImageBlock(p.Image); b != nil {
					blocks = append(blocks, b)
				}
			default:
				blocks = append(blocks, map[string]any{"type": "text", "text": p.Text})
			}
		}
		return blocks
	}
	if m.Content != "" {
		blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
	}
	return blocks
}

func anthropicImageBlock(img *ImageSource) map[string]any {
	if img == nil {
		return nil
	}
	if img.Data != "" {
		mt := img.MediaType
		if mt == "" {
			mt = "application/octet-stream"
		}
		return map[string]any{
			"type": "image",
			"source": map[string]any{
				"type":       "base64",
				"media_type": mt,
				"data":       img.Data,
			},
		}
	}
	if img.URL != "" {
		return map[string]any{
			"type":   "image",
			"source": map[string]any{"type": "url", "url": img.URL},
		}
	}
	return nil
}

func anthropicTools(tools []Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		tool := map[string]any{"name": t.Name}
		if t.Description != "" {
			tool["description"] = t.Description
		}
		if len(t.Parameters) > 0 {
			tool["input_schema"] = json.RawMessage(t.Parameters)
		} else {
			tool["input_schema"] = map[string]any{"type": "object"}
		}
		out = append(out, tool)
	}
	return out
}

func anthropicToolChoice(tc *ToolChoice) any {
	if tc == nil {
		return nil
	}
	switch tc.Mode {
	case ToolChoiceAuto:
		return map[string]any{"type": "auto"}
	case ToolChoiceNone:
		return map[string]any{"type": "none"}
	case ToolChoiceRequired:
		return map[string]any{"type": "any"}
	case ToolChoiceTool:
		return map[string]any{"type": "tool", "name": tc.Name}
	default:
		return nil
	}
}

// DecodeAnthropicRequest parses Anthropic messages JSON into ChatRequest.
func DecodeAnthropicRequest(body []byte) (ChatRequest, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid anthropic messages body: %w", err)
	}
	if err := rejectAnthropicUnsupported(raw); err != nil {
		return ChatRequest{}, err
	}

	var in struct {
		Model       string          `json:"model"`
		Stream      bool            `json:"stream"`
		MaxTokens   int             `json:"max_tokens"`
		Temperature *float64        `json:"temperature"`
		TopP        *float64        `json:"top_p"`
		System      any             `json:"system"`
		Stop        []string        `json:"stop_sequences"`
		Tools       json.RawMessage `json:"tools"`
		ToolChoice  json.RawMessage `json:"tool_choice"`
		Messages    []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid anthropic messages body: %w", err)
	}
	if strings.TrimSpace(in.Model) == "" {
		return ChatRequest{}, fmt.Errorf("model is required")
	}
	req := ChatRequest{
		Model:       in.Model,
		Stream:      in.Stream,
		MaxTokens:   in.MaxTokens,
		Temperature: in.Temperature,
		TopP:        in.TopP,
		Stop:        in.Stop,
		System:      anthropicSystemToString(in.System),
	}
	tools, err := parseAnthropicTools(in.Tools)
	if err != nil {
		return ChatRequest{}, err
	}
	req.Tools = tools
	tc, err := parseAnthropicToolChoice(in.ToolChoice)
	if err != nil {
		return ChatRequest{}, err
	}
	req.ToolChoice = tc

	for _, m := range in.Messages {
		switch m.Role {
		case "assistant":
			text, _, calls, _, err := parseAnthropicContent(m.Content)
			if err != nil {
				return ChatRequest{}, err
			}
			req.Messages = append(req.Messages, Message{Role: "assistant", Content: text, ToolCalls: calls})
		case "user":
			text, parts, _, toolResults, err := parseAnthropicContent(m.Content)
			if err != nil {
				return ChatRequest{}, err
			}
			for _, tr := range toolResults {
				req.Messages = append(req.Messages, Message{Role: "tool", Content: tr.Content, ToolCallID: tr.ToolUseID})
			}
			if text != "" || len(parts) > 0 {
				req.Messages = append(req.Messages, Message{Role: "user", Content: text, Parts: parts})
			}
		}
	}
	if len(req.Messages) == 0 {
		return ChatRequest{}, fmt.Errorf("at least one user/assistant message is required")
	}
	return req, nil
}

// EncodeAnthropicResponse marshals ChatResponse to Anthropic message JSON.
func EncodeAnthropicResponse(resp ChatResponse) ([]byte, error) {
	role := resp.Role
	if role == "" {
		role = "assistant"
	}
	var content []map[string]any
	if resp.Content != "" || len(resp.ToolCalls) == 0 {
		content = append(content, map[string]any{"type": "text", "text": resp.Content})
	}
	for _, tc := range resp.ToolCalls {
		content = append(content, map[string]any{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": argsToObject(tc.Arguments),
		})
	}
	out := map[string]any{
		"id":          resp.ID,
		"type":        "message",
		"role":        role,
		"model":       resp.Model,
		"content":     content,
		"stop_reason": mapFinishToAnthropic(resp.FinishReason),
		"usage": map[string]int64{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
		},
	}
	return json.Marshal(out)
}

// DecodeAnthropicResponse parses Anthropic message JSON into ChatResponse.
func DecodeAnthropicResponse(body []byte) (ChatResponse, error) {
	var in struct {
		ID         string          `json:"id"`
		Model      string          `json:"model"`
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		StopReason string          `json:"stop_reason"`
		Usage      struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatResponse{}, fmt.Errorf("invalid anthropic response: %w", err)
	}
	if in.Error != nil {
		return ChatResponse{}, fmt.Errorf("anthropic: %s", in.Error.Message)
	}
	var content any
	if len(in.Content) > 0 {
		_ = json.Unmarshal(in.Content, &content)
	}
	text, _, calls, _, err := parseAnthropicContent(content)
	if err != nil {
		return ChatResponse{}, err
	}
	role := in.Role
	if role == "" {
		role = "assistant"
	}
	return ChatResponse{
		ID:           in.ID,
		Model:        in.Model,
		Role:         role,
		Content:      text,
		ToolCalls:    calls,
		FinishReason: MapAnthropicStopReason(in.StopReason),
		Usage: Usage{
			PromptTokens:     in.Usage.InputTokens,
			CompletionTokens: in.Usage.OutputTokens,
		},
	}, nil
}

type anthropicToolResult struct {
	ToolUseID string
	Content   string
}

// parseAnthropicContent decomposes Anthropic message content into text, image
// parts, assistant tool_use calls, and user tool_result entries.
func parseAnthropicContent(content any) (text string, parts []ContentPart, calls []ToolCall, results []anthropicToolResult, err error) {
	switch v := content.(type) {
	case string:
		return v, nil, nil, nil, nil
	case nil:
		return "", nil, nil, nil, nil
	case []any:
		var texts []string
		var built []ContentPart
		hasImage := false
		for _, block := range v {
			bm, _ := block.(map[string]any)
			if bm == nil {
				continue
			}
			typ, _ := bm["type"].(string)
			switch typ {
			case "", "text":
				t, _ := bm["text"].(string)
				texts = append(texts, t)
				built = append(built, ContentPart{Type: ContentText, Text: t})
			case "image":
				img := anthropicParseImage(bm)
				if img == nil {
					return "", nil, nil, nil, Unsupported("vision", "image content block has an unsupported source")
				}
				hasImage = true
				built = append(built, ContentPart{Type: ContentImage, Image: img})
			case "tool_use":
				id, _ := bm["id"].(string)
				name, _ := bm["name"].(string)
				calls = append(calls, ToolCall{ID: id, Name: name, Arguments: objectToArgs(bm["input"])})
			case "tool_result":
				id, _ := bm["tool_use_id"].(string)
				results = append(results, anthropicToolResult{ToolUseID: id, Content: anthropicToolResultText(bm["content"])})
			default:
				return "", nil, nil, nil, Unsupported("content_blocks", fmt.Sprintf("content block type %q is not supported by the gateway chat dialect yet", typ))
			}
		}
		text = strings.Join(texts, "")
		if hasImage {
			parts = built
		}
		return text, parts, calls, results, nil
	default:
		return "", nil, nil, nil, Unsupported("content", "message content must be a string or content blocks")
	}
}

func anthropicParseImage(bm map[string]any) *ImageSource {
	src, _ := bm["source"].(map[string]any)
	if src == nil {
		return nil
	}
	styp, _ := src["type"].(string)
	switch styp {
	case "base64":
		data, _ := src["data"].(string)
		if data == "" {
			return nil
		}
		mt, _ := src["media_type"].(string)
		return &ImageSource{MediaType: mt, Data: data}
	case "url":
		url, _ := src["url"].(string)
		if url == "" {
			return nil
		}
		return &ImageSource{URL: url}
	default:
		return nil
	}
}

func anthropicToolResultText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var texts []string
		for _, block := range v {
			bm, _ := block.(map[string]any)
			if bm == nil {
				continue
			}
			if t, ok := bm["text"].(string); ok {
				texts = append(texts, t)
			}
		}
		return strings.Join(texts, "")
	default:
		return ""
	}
}

func parseAnthropicTools(raw json.RawMessage) ([]Tool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var in []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"input_schema"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("invalid tools: %w", err)
	}
	out := make([]Tool, 0, len(in))
	for _, t := range in {
		if t.Name == "" {
			continue
		}
		out = append(out, Tool{Name: t.Name, Description: t.Description, Parameters: t.InputSchema})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func parseAnthropicToolChoice(raw json.RawMessage) (*ToolChoice, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var obj struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("invalid tool_choice: %w", err)
	}
	switch obj.Type {
	case "auto":
		return &ToolChoice{Mode: ToolChoiceAuto}, nil
	case "none":
		return &ToolChoice{Mode: ToolChoiceNone}, nil
	case "any":
		return &ToolChoice{Mode: ToolChoiceRequired}, nil
	case "tool":
		return &ToolChoice{Mode: ToolChoiceTool, Name: obj.Name}, nil
	default:
		return nil, nil
	}
}

// MapAnthropicStopReason maps Anthropic stop_reason to IR finish_reason.
func MapAnthropicStopReason(r string) string {
	switch r {
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "end_turn", "stop_sequence", "":
		return "stop"
	default:
		return r
	}
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

func anthropicSystemToString(system any) string {
	switch v := system.(type) {
	case string:
		return v
	case nil:
		return ""
	case []any:
		var parts []string
		for _, block := range v {
			bm, _ := block.(map[string]any)
			if bm == nil {
				continue
			}
			if t, ok := bm["text"].(string); ok && t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n\n")
	default:
		return contentToString(v)
	}
}

func argsToObject(args string) any {
	if strings.TrimSpace(args) == "" {
		return map[string]any{}
	}
	var v any
	if err := json.Unmarshal([]byte(args), &v); err != nil {
		return map[string]any{}
	}
	return v
}

func objectToArgs(input any) string {
	if input == nil {
		return "{}"
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(b)
}
