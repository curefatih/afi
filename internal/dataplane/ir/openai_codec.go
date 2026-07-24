package ir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EncodeOpenAI marshals a ChatRequest to OpenAI chat.completions JSON.
//
// This is the canonical bridge format used between hooks and provider adapters,
// so it must faithfully carry tools, multimodal content, and tool messages.
func EncodeOpenAI(req ChatRequest) ([]byte, error) {
	msgs := make([]map[string]any, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, map[string]any{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		if m.Role == "system" && req.System != "" {
			// Already lifted to top-level system; skip duplicates when both present.
			continue
		}
		msgs = append(msgs, openAIMessage(m))
	}
	out := map[string]any{
		"model":    req.Model,
		"messages": msgs,
	}
	if req.Stream {
		out["stream"] = true
	}
	if req.MaxTokens > 0 {
		out["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		out["top_p"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		out["stop"] = req.Stop
	}
	if len(req.Tools) > 0 {
		out["tools"] = openAITools(req.Tools)
	}
	if tc := openAIToolChoice(req.ToolChoice); tc != nil {
		out["tool_choice"] = tc
	}
	return json.Marshal(out)
}

func openAIMessage(m Message) map[string]any {
	msg := map[string]any{"role": m.Role}
	switch {
	case len(m.Parts) > 0:
		msg["content"] = openAIContentParts(m.Parts)
	case len(m.ToolCalls) > 0 && m.Content == "":
		msg["content"] = nil
	default:
		msg["content"] = m.Content
	}
	if len(m.ToolCalls) > 0 {
		msg["tool_calls"] = openAIToolCalls(m.ToolCalls)
	}
	if m.Role == "tool" && m.ToolCallID != "" {
		msg["tool_call_id"] = m.ToolCallID
	}
	return msg
}

func openAIContentParts(parts []ContentPart) []map[string]any {
	out := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case ContentImage:
			if url := imageSourceToDataURL(p.Image); url != "" {
				out = append(out, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": url},
				})
			}
		default:
			out = append(out, map[string]any{"type": "text", "text": p.Text})
		}
	}
	return out
}

func openAIToolCalls(calls []ToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, c := range calls {
		args := c.Arguments
		if args == "" {
			args = "{}"
		}
		out = append(out, map[string]any{
			"id":   c.ID,
			"type": "function",
			"function": map[string]any{
				"name":      c.Name,
				"arguments": args,
			},
		})
	}
	return out
}

func openAITools(tools []Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		fn := map[string]any{"name": t.Name}
		if t.Description != "" {
			fn["description"] = t.Description
		}
		if len(t.Parameters) > 0 {
			fn["parameters"] = json.RawMessage(t.Parameters)
		}
		out = append(out, map[string]any{"type": "function", "function": fn})
	}
	return out
}

func openAIToolChoice(tc *ToolChoice) any {
	if tc == nil {
		return nil
	}
	switch tc.Mode {
	case ToolChoiceAuto:
		return "auto"
	case ToolChoiceNone:
		return "none"
	case ToolChoiceRequired:
		return "required"
	case ToolChoiceTool:
		return map[string]any{
			"type":     "function",
			"function": map[string]any{"name": tc.Name},
		}
	default:
		return nil
	}
}

// DecodeOpenAIRequest parses OpenAI chat.completions JSON into ChatRequest.
func DecodeOpenAIRequest(body []byte) (ChatRequest, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid openai chat body: %w", err)
	}
	if err := rejectOpenAIUnsupported(raw); err != nil {
		return ChatRequest{}, err
	}

	var in struct {
		Model               string          `json:"model"`
		Stream              bool            `json:"stream"`
		MaxTokens           *int            `json:"max_tokens"`
		MaxCompletionTokens *int            `json:"max_completion_tokens"`
		Temperature         *float64        `json:"temperature"`
		TopP                *float64        `json:"top_p"`
		Stop                any             `json:"stop"`
		Tools               json.RawMessage `json:"tools"`
		ToolChoice          json.RawMessage `json:"tool_choice"`
		Messages            []struct {
			Role       string          `json:"role"`
			Content    any             `json:"content"`
			ToolCallID string          `json:"tool_call_id"`
			ToolCalls  json.RawMessage `json:"tool_calls"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid openai chat body: %w", err)
	}
	if strings.TrimSpace(in.Model) == "" {
		return ChatRequest{}, fmt.Errorf("model is required")
	}
	req := ChatRequest{Model: in.Model, Stream: in.Stream, Temperature: in.Temperature, TopP: in.TopP}
	if in.MaxTokens != nil && *in.MaxTokens > 0 {
		req.MaxTokens = *in.MaxTokens
	} else if in.MaxCompletionTokens != nil && *in.MaxCompletionTokens > 0 {
		req.MaxTokens = *in.MaxCompletionTokens
	}
	req.Stop = normalizeStop(in.Stop)

	tools, err := parseOpenAITools(in.Tools)
	if err != nil {
		return ChatRequest{}, err
	}
	req.Tools = tools
	tc, err := parseOpenAIToolChoice(in.ToolChoice)
	if err != nil {
		return ChatRequest{}, err
	}
	req.ToolChoice = tc

	var systemParts []string
	for _, m := range in.Messages {
		text, parts, err := parseOpenAIContent(m.Content)
		if err != nil {
			return ChatRequest{}, err
		}
		toolCalls, err := parseOpenAIToolCalls(m.ToolCalls)
		if err != nil {
			return ChatRequest{}, err
		}
		switch m.Role {
		case "system":
			if text != "" {
				systemParts = append(systemParts, text)
			}
		case "user", "assistant":
			req.Messages = append(req.Messages, Message{Role: m.Role, Content: text, Parts: parts, ToolCalls: toolCalls})
		case "tool":
			req.Messages = append(req.Messages, Message{Role: "tool", Content: text, ToolCallID: m.ToolCallID})
		default:
			if text != "" || len(toolCalls) > 0 {
				req.Messages = append(req.Messages, Message{Role: m.Role, Content: text, Parts: parts, ToolCalls: toolCalls})
			}
		}
	}
	if len(systemParts) > 0 {
		req.System = strings.Join(systemParts, "\n\n")
	}
	if len(req.Messages) == 0 {
		return ChatRequest{}, fmt.Errorf("at least one user/assistant message is required")
	}
	return req, nil
}

// EncodeOpenAIResponse marshals ChatResponse to OpenAI chat.completion JSON.
func EncodeOpenAIResponse(resp ChatResponse) ([]byte, error) {
	role := resp.Role
	if role == "" {
		role = "assistant"
	}
	message := map[string]any{"role": role}
	if resp.Content != "" || len(resp.ToolCalls) == 0 {
		message["content"] = resp.Content
	} else {
		message["content"] = nil
	}
	if len(resp.ToolCalls) > 0 {
		message["tool_calls"] = openAIToolCalls(resp.ToolCalls)
	}
	out := map[string]any{
		"id":     resp.ID,
		"object": "chat.completion",
		"model":  resp.Model,
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       message,
				"finish_reason": resp.FinishReason,
			},
		},
		"usage": map[string]int64{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.PromptTokens + resp.Usage.CompletionTokens,
		},
	}
	return json.Marshal(out)
}

// DecodeOpenAIResponse parses OpenAI chat.completion JSON into ChatResponse.
func DecodeOpenAIResponse(body []byte) (ChatResponse, error) {
	var in struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role      string          `json:"role"`
				Content   any             `json:"content"`
				ToolCalls json.RawMessage `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatResponse{}, fmt.Errorf("invalid openai chat response: %w", err)
	}
	if in.Error != nil {
		return ChatResponse{}, fmt.Errorf("openai: %s", in.Error.Message)
	}
	if len(in.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("openai: empty choices")
	}
	c := in.Choices[0]
	toolCalls, err := parseOpenAIToolCalls(c.Message.ToolCalls)
	if err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{
		ID:           in.ID,
		Model:        in.Model,
		Role:         c.Message.Role,
		Content:      contentToString(c.Message.Content),
		ToolCalls:    toolCalls,
		FinishReason: c.FinishReason,
		Usage: Usage{
			PromptTokens:     in.Usage.PromptTokens,
			CompletionTokens: in.Usage.CompletionTokens,
		},
	}, nil
}

// parseOpenAIContent extracts plain text and multimodal parts from OpenAI
// message content (string, or an array of text/image_url parts).
func parseOpenAIContent(content any) (text string, parts []ContentPart, err error) {
	switch v := content.(type) {
	case string:
		return v, nil, nil
	case nil:
		return "", nil, nil
	case []any:
		var texts []string
		var built []ContentPart
		hasImage := false
		for _, part := range v {
			pm, _ := part.(map[string]any)
			if pm == nil {
				continue
			}
			typ, _ := pm["type"].(string)
			switch typ {
			case "", "text":
				t, _ := pm["text"].(string)
				texts = append(texts, t)
				built = append(built, ContentPart{Type: ContentText, Text: t})
			case "image_url":
				img := openAIImagePart(pm)
				if img == nil {
					return "", nil, Unsupported("vision", "image_url content part is missing a url")
				}
				hasImage = true
				built = append(built, ContentPart{Type: ContentImage, Image: img})
			default:
				return "", nil, Unsupported("content_parts", fmt.Sprintf("content part type %q is not supported by the gateway chat dialect yet", typ))
			}
		}
		text = strings.Join(texts, "\n")
		if hasImage {
			parts = built
		}
		return text, parts, nil
	default:
		return "", nil, Unsupported("content", "message content must be a string or content parts")
	}
}

func openAIImagePart(pm map[string]any) *ImageSource {
	iu, ok := pm["image_url"]
	if !ok {
		return nil
	}
	switch v := iu.(type) {
	case string:
		return parseImageURL(v)
	case map[string]any:
		url, _ := v["url"].(string)
		return parseImageURL(url)
	default:
		return nil
	}
}

func parseOpenAIToolCalls(raw json.RawMessage) ([]ToolCall, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var in []struct {
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("invalid tool_calls: %w", err)
	}
	out := make([]ToolCall, 0, len(in))
	for _, c := range in {
		out = append(out, ToolCall{ID: c.ID, Name: c.Function.Name, Arguments: c.Function.Arguments})
	}
	return out, nil
}

func parseOpenAITools(raw json.RawMessage) ([]Tool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var in []struct {
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("invalid tools: %w", err)
	}
	out := make([]Tool, 0, len(in))
	for _, t := range in {
		if t.Function.Name == "" {
			continue
		}
		out = append(out, Tool{Name: t.Function.Name, Description: t.Function.Description, Parameters: t.Function.Parameters})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func parseOpenAIToolChoice(raw json.RawMessage) (*ToolChoice, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch s {
		case "auto":
			return &ToolChoice{Mode: ToolChoiceAuto}, nil
		case "none":
			return &ToolChoice{Mode: ToolChoiceNone}, nil
		case "required", "any":
			return &ToolChoice{Mode: ToolChoiceRequired}, nil
		default:
			return nil, nil
		}
	}
	var obj struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("invalid tool_choice: %w", err)
	}
	if obj.Function.Name != "" {
		return &ToolChoice{Mode: ToolChoiceTool, Name: obj.Function.Name}, nil
	}
	return nil, nil
}

func contentToString(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	case []any:
		var texts []string
		for _, part := range v {
			pm, _ := part.(map[string]any)
			if pm == nil {
				continue
			}
			if t, ok := pm["text"].(string); ok {
				texts = append(texts, t)
			}
		}
		return strings.Join(texts, "\n")
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func normalizeStop(stop any) []string {
	switch v := stop.(type) {
	case nil:
		return nil
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
