package dialect

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

// Gemini is the Google Gemini generateContent client dialect codec.
type Gemini struct{}

func (Gemini) Name() ir.Dialect { return ir.DialectGemini }

func (Gemini) DecodeRequest(body []byte) (ir.ChatRequest, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return ir.ChatRequest{}, fmt.Errorf("invalid gemini generateContent body: %w", err)
	}
	if err := rejectGeminiUnsupported(raw); err != nil {
		return ir.ChatRequest{}, err
	}

	req := ir.ChatRequest{}
	if sys, ok := raw["systemInstruction"].(map[string]any); ok {
		text, _, _, _, err := decodeGeminiParts(sys["parts"], nil)
		if err != nil {
			return ir.ChatRequest{}, err
		}
		req.System = text
	}
	if cfg, ok := raw["generationConfig"].(map[string]any); ok {
		req.MaxTokens = intFromJSON(cfg["maxOutputTokens"])
		req.Temperature = floatPtr(cfg["temperature"])
		req.TopP = floatPtr(cfg["topP"])
		req.Stop = stringSlice(cfg["stopSequences"])
	}

	tools, err := decodeGeminiTools(raw["tools"])
	if err != nil {
		return ir.ChatRequest{}, err
	}
	req.Tools = tools
	req.ToolChoice = decodeGeminiToolChoice(raw["toolConfig"])

	callIDs := map[string][]string{}
	callNumber := 0
	usedResult := map[string]int{}
	contents, _ := raw["contents"].([]any)
	for _, item := range contents {
		content, _ := item.(map[string]any)
		if content == nil {
			continue
		}
		role, _ := content["role"].(string)
		text, parts, calls, results, err := decodeGeminiParts(content["parts"], func(name, id string) string {
			if id == "" {
				id = fmt.Sprintf("call_gemini_%d", callNumber)
			}
			callNumber++
			callIDs[name] = append(callIDs[name], id)
			return id
		})
		if err != nil {
			return ir.ChatRequest{}, err
		}
		switch role {
		case "model":
			req.Messages = append(req.Messages, ir.Message{
				Role: "assistant", Content: text, Parts: parts, ToolCalls: calls,
			})
		case "user", "":
			for _, result := range results {
				id := result.id
				if id == "" {
					ids := callIDs[result.name]
					idx := usedResult[result.name]
					if idx < len(ids) {
						id = ids[idx]
						usedResult[result.name] = idx + 1
					}
				}
				if id == "" {
					id = fmt.Sprintf("call_gemini_%d", callNumber)
					callNumber++
				}
				req.Messages = append(req.Messages, ir.Message{
					Role: "tool", Content: result.content, ToolCallID: id,
				})
			}
			if text != "" || len(parts) > 0 {
				req.Messages = append(req.Messages, ir.Message{
					Role: "user", Content: text, Parts: parts,
				})
			}
		default:
			return ir.ChatRequest{}, ir.Unsupported("role", fmt.Sprintf("gemini role %q is not supported", role))
		}
	}
	if len(req.Messages) == 0 {
		return ir.ChatRequest{}, fmt.Errorf("at least one user/model content is required")
	}
	return req, nil
}

// EncodeGeminiRequest marshals chat IR to a Gemini generateContent request.
// Provider adapters use the same mapping as the client dialect.
func EncodeGeminiRequest(req ir.ChatRequest) ([]byte, error) {
	var contents []any
	callNames := map[string]string{}
	for _, message := range req.Messages {
		for _, call := range message.ToolCalls {
			callNames[call.ID] = call.Name
		}
	}
	for _, message := range req.Messages {
		switch message.Role {
		case "system":
			continue
		case "user", "assistant":
			role := "user"
			if message.Role == "assistant" {
				role = "model"
			}
			parts := encodeGeminiContentParts(message)
			for _, call := range message.ToolCalls {
				parts = append(parts, map[string]any{"functionCall": map[string]any{
					"id": call.ID, "name": call.Name, "args": geminiArgsObject(call.Arguments),
				}})
			}
			if len(parts) > 0 {
				contents = append(contents, map[string]any{"role": role, "parts": parts})
			}
		case "tool":
			name := callNames[message.ToolCallID]
			if name == "" {
				return nil, ir.Unsupported(
					"tool_result",
					fmt.Sprintf("tool result %q has no matching tool call", message.ToolCallID),
				)
			}
			contents = append(contents, map[string]any{
				"role": "user",
				"parts": []any{map[string]any{"functionResponse": map[string]any{
					"id": message.ToolCallID, "name": name,
					"response": geminiResponseObject(message.Content),
				}}},
			})
		default:
			return nil, ir.Unsupported("role", fmt.Sprintf("message role %q cannot be mapped to Gemini", message.Role))
		}
	}
	if len(contents) == 0 {
		return nil, fmt.Errorf("at least one user/model content is required")
	}

	out := map[string]any{"contents": contents}
	if req.System != "" {
		out["systemInstruction"] = map[string]any{
			"parts": []any{map[string]any{"text": req.System}},
		}
	}
	cfg := map[string]any{}
	if req.MaxTokens > 0 {
		cfg["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		cfg["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		cfg["topP"] = *req.TopP
	}
	if len(req.Stop) > 0 {
		cfg["stopSequences"] = req.Stop
	}
	if len(cfg) > 0 {
		out["generationConfig"] = cfg
	}
	if len(req.Tools) > 0 {
		declarations := make([]any, 0, len(req.Tools))
		for _, tool := range req.Tools {
			declaration := map[string]any{"name": tool.Name}
			if tool.Description != "" {
				declaration["description"] = tool.Description
			}
			if len(tool.Parameters) > 0 {
				declaration["parameters"] = json.RawMessage(tool.Parameters)
			} else {
				declaration["parameters"] = map[string]any{"type": "object"}
			}
			declarations = append(declarations, declaration)
		}
		out["tools"] = []any{map[string]any{"functionDeclarations": declarations}}
	}
	if req.ToolChoice != nil {
		functionConfig := map[string]any{}
		switch req.ToolChoice.Mode {
		case ir.ToolChoiceAuto:
			functionConfig["mode"] = "AUTO"
		case ir.ToolChoiceNone:
			functionConfig["mode"] = "NONE"
		case ir.ToolChoiceRequired:
			functionConfig["mode"] = "ANY"
		case ir.ToolChoiceTool:
			functionConfig["mode"] = "ANY"
			functionConfig["allowedFunctionNames"] = []string{req.ToolChoice.Name}
		}
		if len(functionConfig) > 0 {
			out["toolConfig"] = map[string]any{"functionCallingConfig": functionConfig}
		}
	}
	return json.Marshal(out)
}

func (Gemini) EncodeResponse(resp ir.ChatResponse) ([]byte, error) {
	parts := make([]any, 0, 1+len(resp.ToolCalls))
	if resp.Content != "" || len(resp.ToolCalls) == 0 {
		parts = append(parts, map[string]any{"text": resp.Content})
	}
	for _, call := range resp.ToolCalls {
		parts = append(parts, map[string]any{
			"functionCall": map[string]any{
				"id": call.ID, "name": call.Name, "args": geminiArgsObject(call.Arguments),
			},
		})
	}
	out := map[string]any{
		"responseId":   resp.ID,
		"modelVersion": resp.Model,
		"candidates": []any{map[string]any{
			"index":        0,
			"content":      map[string]any{"role": "model", "parts": parts},
			"finishReason": geminiFinishReason(resp.FinishReason),
		}},
		"usageMetadata": map[string]int64{
			"promptTokenCount":     resp.Usage.PromptTokens,
			"candidatesTokenCount": resp.Usage.CompletionTokens,
			"totalTokenCount":      resp.Usage.PromptTokens + resp.Usage.CompletionTokens,
		},
	}
	return json.Marshal(out)
}

func encodeGeminiContentParts(message ir.Message) []any {
	var parts []any
	if len(message.Parts) == 0 {
		if message.Content != "" {
			parts = append(parts, map[string]any{"text": message.Content})
		}
		return parts
	}
	for _, part := range message.Parts {
		switch part.Type {
		case ir.ContentImage:
			if part.Image == nil {
				continue
			}
			if part.Image.Data != "" {
				parts = append(parts, map[string]any{"inlineData": map[string]any{
					"mimeType": part.Image.MediaType, "data": part.Image.Data,
				}})
			} else if part.Image.URL != "" {
				parts = append(parts, map[string]any{"fileData": map[string]any{
					"mimeType": part.Image.MediaType, "fileUri": part.Image.URL,
				}})
			}
		default:
			parts = append(parts, map[string]any{"text": part.Text})
		}
	}
	return parts
}

func (Gemini) WriteStream(w io.Writer, events <-chan ir.StreamEvent) (prompt, completion int64, err error) {
	flusher, _ := w.(http.Flusher)
	type pendingTool struct {
		id, name string
		args     strings.Builder
	}
	tools := map[int]*pendingTool{}
	var order []int

	write := func(payload any) error {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err = fmt.Fprintf(w, "data: %s\n\n", body); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}
	writeCandidate := func(parts []any, finish string, usage *ir.Usage) error {
		candidate := map[string]any{
			"index":   0,
			"content": map[string]any{"role": "model", "parts": parts},
		}
		if finish != "" {
			candidate["finishReason"] = geminiFinishReason(finish)
		}
		payload := map[string]any{"candidates": []any{candidate}}
		if usage != nil {
			prompt, completion = usage.PromptTokens, usage.CompletionTokens
			payload["usageMetadata"] = map[string]int64{
				"promptTokenCount": prompt, "candidatesTokenCount": completion,
				"totalTokenCount": prompt + completion,
			}
		}
		return write(payload)
	}

	for event := range events {
		switch event.Kind {
		case ir.StreamError:
			if event.Err != nil {
				return prompt, completion, event.Err
			}
			return prompt, completion, fmt.Errorf("stream error")
		case ir.StreamTextDelta:
			if event.Text != "" {
				if err := writeCandidate([]any{map[string]any{"text": event.Text}}, "", nil); err != nil {
					return prompt, completion, err
				}
			}
		case ir.StreamToolCallStart:
			if _, exists := tools[event.ToolIndex]; !exists {
				tools[event.ToolIndex] = &pendingTool{id: event.ToolID, name: event.ToolName}
				order = append(order, event.ToolIndex)
			}
		case ir.StreamToolCallDelta:
			if tool := tools[event.ToolIndex]; tool != nil {
				tool.args.WriteString(event.ArgsDelta)
			}
		case ir.StreamMessageEnd:
			if len(order) > 0 {
				parts := make([]any, 0, len(order))
				for _, idx := range order {
					tool := tools[idx]
					parts = append(parts, map[string]any{"functionCall": map[string]any{
						"id": tool.id, "name": tool.name, "args": geminiArgsObject(tool.args.String()),
					}})
				}
				if err := writeCandidate(parts, "", nil); err != nil {
					return prompt, completion, err
				}
				order = nil
				tools = map[int]*pendingTool{}
			}
			if err := writeCandidate([]any{}, event.FinishReason, event.Usage); err != nil {
				return prompt, completion, err
			}
		}
	}
	return prompt, completion, nil
}

type geminiToolResult struct {
	id, name, content string
}

func decodeGeminiParts(
	raw any,
	registerCall func(name, id string) string,
) (text string, parts []ir.ContentPart, calls []ir.ToolCall, results []geminiToolResult, err error) {
	items, _ := raw.([]any)
	var texts []string
	var built []ir.ContentPart
	hasImage := false
	for _, item := range items {
		part, _ := item.(map[string]any)
		if part == nil {
			continue
		}
		switch {
		case part["text"] != nil:
			value, _ := part["text"].(string)
			texts = append(texts, value)
			built = append(built, ir.ContentPart{Type: ir.ContentText, Text: value})
		case part["inlineData"] != nil:
			source, _ := part["inlineData"].(map[string]any)
			mediaType, _ := source["mimeType"].(string)
			data, _ := source["data"].(string)
			if data == "" {
				return "", nil, nil, nil, ir.Unsupported("vision", "inlineData image is missing data")
			}
			hasImage = true
			built = append(built, ir.ContentPart{Type: ir.ContentImage, Image: &ir.ImageSource{
				MediaType: mediaType, Data: data,
			}})
		case part["fileData"] != nil:
			source, _ := part["fileData"].(map[string]any)
			uri, _ := source["fileUri"].(string)
			mediaType, _ := source["mimeType"].(string)
			if uri == "" {
				return "", nil, nil, nil, ir.Unsupported("vision", "fileData image is missing fileUri")
			}
			hasImage = true
			built = append(built, ir.ContentPart{Type: ir.ContentImage, Image: &ir.ImageSource{
				URL: uri, MediaType: mediaType,
			}})
		case part["functionCall"] != nil:
			call, _ := part["functionCall"].(map[string]any)
			name, _ := call["name"].(string)
			id, _ := call["id"].(string)
			if registerCall != nil {
				id = registerCall(name, id)
			}
			calls = append(calls, ir.ToolCall{ID: id, Name: name, Arguments: geminiArgsJSON(call["args"])})
		case part["functionResponse"] != nil:
			response, _ := part["functionResponse"].(map[string]any)
			id, _ := response["id"].(string)
			name, _ := response["name"].(string)
			results = append(results, geminiToolResult{
				id: id, name: name, content: geminiArgsJSON(response["response"]),
			})
		default:
			return "", nil, nil, nil, ir.Unsupported("content_parts", "gemini content part type is not supported by the gateway")
		}
	}
	text = strings.Join(texts, "")
	if hasImage {
		parts = built
	}
	return text, parts, calls, results, nil
}

func decodeGeminiTools(raw any) ([]ir.Tool, error) {
	groups, _ := raw.([]any)
	var out []ir.Tool
	for _, groupRaw := range groups {
		group, _ := groupRaw.(map[string]any)
		declarations, _ := group["functionDeclarations"].([]any)
		if len(declarations) == 0 && len(group) > 0 {
			return nil, ir.Unsupported("tools", "only Gemini functionDeclarations tools are supported")
		}
		for _, declarationRaw := range declarations {
			declaration, _ := declarationRaw.(map[string]any)
			name, _ := declaration["name"].(string)
			description, _ := declaration["description"].(string)
			schema := declaration["parameters"]
			if schema == nil {
				schema = declaration["parametersJsonSchema"]
			}
			encoded, _ := json.Marshal(schema)
			out = append(out, ir.Tool{Name: name, Description: description, Parameters: encoded})
		}
	}
	return out, nil
}

func decodeGeminiToolChoice(raw any) *ir.ToolChoice {
	config, _ := raw.(map[string]any)
	functionConfig, _ := config["functionCallingConfig"].(map[string]any)
	mode, _ := functionConfig["mode"].(string)
	switch strings.ToUpper(mode) {
	case "AUTO":
		return &ir.ToolChoice{Mode: ir.ToolChoiceAuto}
	case "NONE":
		return &ir.ToolChoice{Mode: ir.ToolChoiceNone}
	case "ANY":
		names := stringSlice(functionConfig["allowedFunctionNames"])
		if len(names) == 1 {
			return &ir.ToolChoice{Mode: ir.ToolChoiceTool, Name: names[0]}
		}
		return &ir.ToolChoice{Mode: ir.ToolChoiceRequired}
	default:
		return nil
	}
}

func geminiArgsJSON(value any) string {
	if value == nil {
		return "{}"
	}
	body, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(body)
}

func geminiArgsObject(value string) any {
	if strings.TrimSpace(value) == "" {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func geminiResponseObject(value string) map[string]any {
	var object map[string]any
	if json.Unmarshal([]byte(value), &object) == nil && object != nil {
		return object
	}
	return map[string]any{"result": value}
}

func geminiFinishReason(reason string) string {
	switch reason {
	case "length":
		return "MAX_TOKENS"
	case "stop", "tool_calls", "":
		return "STOP"
	default:
		return strings.ToUpper(reason)
	}
}

func rejectGeminiUnsupported(raw map[string]any) error {
	if raw == nil {
		return nil
	}
	if v, ok := raw["cachedContent"]; ok && !isJSONNull(v) {
		return ir.Unsupported("cached_content", "cachedContent is not supported by the gateway chat dialect yet")
	}
	if v, ok := raw["safetySettings"]; ok && !isJSONNull(v) {
		return ir.Unsupported("safety_settings", "safetySettings are not supported by the gateway chat dialect yet")
	}
	if cfg, ok := raw["generationConfig"].(map[string]any); ok {
		for _, field := range []string{
			"responseMimeType", "responseSchema", "responseJsonSchema",
			"thinkingConfig", "seed", "logprobs", "responseLogprobs",
			"presencePenalty", "frequencyPenalty", "audioTimestamp",
		} {
			if v, exists := cfg[field]; exists && !isJSONNull(v) {
				return ir.Unsupported(field, fmt.Sprintf("generationConfig.%s is not supported by the gateway chat dialect yet", field))
			}
		}
		if n := intFromJSON(cfg["candidateCount"]); n > 1 {
			return ir.Unsupported("candidate_count", "candidateCount > 1 is not supported by the gateway chat dialect")
		}
	}
	return nil
}

func isJSONNull(v any) bool {
	return v == nil
}

func intFromJSON(value any) int {
	number, _ := value.(float64)
	return int(number)
}

func floatPtr(value any) *float64 {
	number, ok := value.(float64)
	if !ok {
		return nil
	}
	return &number
}

func stringSlice(value any) []string {
	items, _ := value.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}
