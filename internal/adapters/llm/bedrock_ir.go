package llm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/curefatih/afi/internal/dataplane/ir"
)

// irToConverseInput maps chat IR to a Bedrock Converse request.
func irToConverseInput(targetModel string, req ir.ChatRequest) (*bedrockruntime.ConverseInput, error) {
	msgs, err := irToConverseMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	in := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(targetModel),
		Messages: msgs,
	}
	if req.System != "" {
		in.System = []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: req.System},
		}
	}
	if cfg := irToInferenceConfig(req); cfg != nil {
		in.InferenceConfig = cfg
	}
	if tc, err := irToToolConfig(req); err != nil {
		return nil, err
	} else if tc != nil {
		in.ToolConfig = tc
	}
	return in, nil
}

// irToConverseStreamInput maps chat IR to a Bedrock ConverseStream request.
func irToConverseStreamInput(targetModel string, req ir.ChatRequest) (*bedrockruntime.ConverseStreamInput, error) {
	base, err := irToConverseInput(targetModel, req)
	if err != nil {
		return nil, err
	}
	return &bedrockruntime.ConverseStreamInput{
		ModelId:                           base.ModelId,
		Messages:                          base.Messages,
		System:                            base.System,
		InferenceConfig:                   base.InferenceConfig,
		ToolConfig:                        base.ToolConfig,
		AdditionalModelRequestFields:      base.AdditionalModelRequestFields,
		AdditionalModelResponseFieldPaths: base.AdditionalModelResponseFieldPaths,
	}, nil
}

func irToInferenceConfig(req ir.ChatRequest) *types.InferenceConfiguration {
	if req.MaxTokens <= 0 && req.Temperature == nil && req.TopP == nil && len(req.Stop) == 0 {
		return nil
	}
	cfg := &types.InferenceConfiguration{}
	if req.MaxTokens > 0 {
		cfg.MaxTokens = aws.Int32(int32(req.MaxTokens))
	}
	if req.Temperature != nil {
		t := float32(*req.Temperature)
		cfg.Temperature = &t
	}
	if req.TopP != nil {
		p := float32(*req.TopP)
		cfg.TopP = &p
	}
	if len(req.Stop) > 0 {
		cfg.StopSequences = append([]string(nil), req.Stop...)
	}
	return cfg
}

func irToToolConfig(req ir.ChatRequest) (*types.ToolConfiguration, error) {
	if len(req.Tools) == 0 && req.ToolChoice == nil {
		return nil, nil
	}
	tools := make([]types.Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		var schema any = map[string]any{"type": "object", "properties": map[string]any{}}
		if len(t.Parameters) > 0 {
			if err := json.Unmarshal(t.Parameters, &schema); err != nil {
				return nil, fmt.Errorf("tool %q parameters: %w", t.Name, err)
			}
		}
		spec := types.ToolSpecification{
			Name:        aws.String(t.Name),
			InputSchema: &types.ToolInputSchemaMemberJson{Value: document.NewLazyDocument(schema)},
		}
		if t.Description != "" {
			spec.Description = aws.String(t.Description)
		}
		tools = append(tools, &types.ToolMemberToolSpec{Value: spec})
	}
	if req.ToolChoice != nil && req.ToolChoice.Mode == ir.ToolChoiceNone {
		// Bedrock has no explicit "none"; omit tools so the model cannot call them.
		return nil, nil
	}
	if len(tools) == 0 {
		return nil, nil
	}
	cfg := &types.ToolConfiguration{Tools: tools}
	if req.ToolChoice != nil {
		switch req.ToolChoice.Mode {
		case ir.ToolChoiceAuto:
			cfg.ToolChoice = &types.ToolChoiceMemberAuto{Value: types.AutoToolChoice{}}
		case ir.ToolChoiceRequired:
			cfg.ToolChoice = &types.ToolChoiceMemberAny{Value: types.AnyToolChoice{}}
		case ir.ToolChoiceTool:
			cfg.ToolChoice = &types.ToolChoiceMemberTool{
				Value: types.SpecificToolChoice{Name: aws.String(req.ToolChoice.Name)},
			}
		}
	}
	return cfg, nil
}

func irToConverseMessages(messages []ir.Message) ([]types.Message, error) {
	out := make([]types.Message, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "system":
			// Top-level System is preferred; skip message-level system here.
			continue
		case "user", "tool":
			content, err := irUserContent(m)
			if err != nil {
				return nil, err
			}
			if len(content) == 0 {
				continue
			}
			out = append(out, types.Message{
				Role:    types.ConversationRoleUser,
				Content: content,
			})
		case "assistant":
			content, err := irAssistantContent(m)
			if err != nil {
				return nil, err
			}
			if len(content) == 0 {
				continue
			}
			out = append(out, types.Message{
				Role:    types.ConversationRoleAssistant,
				Content: content,
			})
		default:
			return nil, ir.Unsupported("role", "unsupported message role \""+m.Role+"\" for Bedrock Converse")
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one user/assistant message is required")
	}
	return out, nil
}

func irUserContent(m ir.Message) ([]types.ContentBlock, error) {
	if m.Role == "tool" || m.ToolCallID != "" {
		return []types.ContentBlock{
			&types.ContentBlockMemberToolResult{
				Value: types.ToolResultBlock{
					ToolUseId: aws.String(m.ToolCallID),
					Content: []types.ToolResultContentBlock{
						&types.ToolResultContentBlockMemberText{Value: m.Content},
					},
				},
			},
		}, nil
	}
	return irContentBlocks(m)
}

func irAssistantContent(m ir.Message) ([]types.ContentBlock, error) {
	blocks, err := irContentBlocks(m)
	if err != nil {
		return nil, err
	}
	for _, call := range m.ToolCalls {
		var input any = map[string]any{}
		if call.Arguments != "" {
			if err := json.Unmarshal([]byte(call.Arguments), &input); err != nil {
				// Pass raw string wrapped if not valid JSON object.
				input = map[string]any{"raw": call.Arguments}
			}
		}
		blocks = append(blocks, &types.ContentBlockMemberToolUse{
			Value: types.ToolUseBlock{
				ToolUseId: aws.String(call.ID),
				Name:      aws.String(call.Name),
				Input:     document.NewLazyDocument(input),
			},
		})
	}
	return blocks, nil
}

func irContentBlocks(m ir.Message) ([]types.ContentBlock, error) {
	var blocks []types.ContentBlock
	if len(m.Parts) > 0 {
		for _, p := range m.Parts {
			switch p.Type {
			case ir.ContentText:
				if p.Text != "" {
					blocks = append(blocks, &types.ContentBlockMemberText{Value: p.Text})
				}
			case ir.ContentImage:
				img, err := irImageBlock(p.Image)
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, &types.ContentBlockMemberImage{Value: *img})
			default:
				return nil, ir.Unsupported("content", "unsupported content part type for Bedrock")
			}
		}
		return blocks, nil
	}
	if m.Content != "" {
		blocks = append(blocks, &types.ContentBlockMemberText{Value: m.Content})
	}
	return blocks, nil
}

func irImageBlock(img *ir.ImageSource) (*types.ImageBlock, error) {
	if img == nil {
		return nil, ir.Unsupported("vision", "image part is missing source")
	}
	if img.Data == "" {
		return nil, ir.Unsupported("vision", "Bedrock Converse requires inline base64 image data (URL-only images are not supported)")
	}
	raw, err := base64.StdEncoding.DecodeString(img.Data)
	if err != nil {
		// Try raw encoding without padding variants.
		raw, err = base64.RawStdEncoding.DecodeString(img.Data)
		if err != nil {
			return nil, fmt.Errorf("invalid image base64: %w", err)
		}
	}
	format, err := mediaTypeToImageFormat(img.MediaType)
	if err != nil {
		return nil, err
	}
	return &types.ImageBlock{
		Format: format,
		Source: &types.ImageSourceMemberBytes{Value: raw},
	}, nil
}

func mediaTypeToImageFormat(mediaType string) (types.ImageFormat, error) {
	mt := strings.ToLower(strings.TrimSpace(mediaType))
	switch mt {
	case "image/png", "png", "":
		if mt == "" {
			return types.ImageFormatPng, nil
		}
		return types.ImageFormatPng, nil
	case "image/jpeg", "image/jpg", "jpeg", "jpg":
		return types.ImageFormatJpeg, nil
	case "image/gif", "gif":
		return types.ImageFormatGif, nil
	case "image/webp", "webp":
		return types.ImageFormatWebp, nil
	default:
		return "", ir.Unsupported("vision", "unsupported image media type \""+mediaType+"\" for Bedrock")
	}
}

func converseOutputToIR(out *bedrockruntime.ConverseOutput, model string) (ir.ChatResponse, error) {
	if out == nil {
		return ir.ChatResponse{}, fmt.Errorf("empty bedrock converse response")
	}
	msg, ok := out.Output.(*types.ConverseOutputMemberMessage)
	if !ok || msg == nil {
		return ir.ChatResponse{}, fmt.Errorf("bedrock converse: unexpected output type")
	}
	text, toolCalls, err := converseContentToIR(msg.Value.Content)
	if err != nil {
		return ir.ChatResponse{}, err
	}
	finish := mapBedrockStopReason(out.StopReason)
	if len(toolCalls) > 0 && finish == "stop" {
		finish = "tool_calls"
	}
	resp := ir.ChatResponse{
		ID:           "chatcmpl-bedrock",
		Model:        model,
		Role:         "assistant",
		Content:      text,
		ToolCalls:    toolCalls,
		FinishReason: finish,
	}
	if out.Usage != nil {
		if out.Usage.InputTokens != nil {
			resp.Usage.PromptTokens = int64(*out.Usage.InputTokens)
		}
		if out.Usage.OutputTokens != nil {
			resp.Usage.CompletionTokens = int64(*out.Usage.OutputTokens)
		}
	}
	return resp, nil
}

func converseContentToIR(blocks []types.ContentBlock) (string, []ir.ToolCall, error) {
	var text strings.Builder
	var calls []ir.ToolCall
	for _, b := range blocks {
		switch v := b.(type) {
		case *types.ContentBlockMemberText:
			text.WriteString(v.Value)
		case *types.ContentBlockMemberToolUse:
			args := "{}"
			if v.Value.Input != nil {
				var raw any
				if err := v.Value.Input.UnmarshalSmithyDocument(&raw); err == nil {
					if buf, err := json.Marshal(raw); err == nil {
						args = string(buf)
					}
				}
			}
			id := aws.ToString(v.Value.ToolUseId)
			name := aws.ToString(v.Value.Name)
			calls = append(calls, ir.ToolCall{ID: id, Name: name, Arguments: args})
		}
	}
	return text.String(), calls, nil
}

func mapBedrockStopReason(r types.StopReason) string {
	switch r {
	case types.StopReasonMaxTokens:
		return "length"
	case types.StopReasonToolUse:
		return "tool_calls"
	case types.StopReasonEndTurn, types.StopReasonStopSequence, "":
		return "stop"
	default:
		return string(r)
	}
}

// mapConverseStreamEvent converts a single Bedrock stream event into zero or more IR events.
func mapConverseStreamEvent(
	ev types.ConverseStreamOutput,
	model string,
	id string,
	nextToolIndex *int,
	toolIndexes map[int]int, // contentBlockIndex -> tool index
) []ir.StreamEvent {
	switch v := ev.(type) {
	case *types.ConverseStreamOutputMemberMessageStart:
		return []ir.StreamEvent{{
			Kind: ir.StreamMessageStart, ID: id, Model: model, Role: "assistant",
		}}
	case *types.ConverseStreamOutputMemberContentBlockStart:
		start := v.Value.Start
		if toolStart, ok := start.(*types.ContentBlockStartMemberToolUse); ok {
			idx := *nextToolIndex
			*nextToolIndex++
			if v.Value.ContentBlockIndex != nil {
				toolIndexes[int(*v.Value.ContentBlockIndex)] = idx
			}
			return []ir.StreamEvent{{
				Kind: ir.StreamToolCallStart, ID: id, Model: model,
				ToolIndex: idx,
				ToolID:    aws.ToString(toolStart.Value.ToolUseId),
				ToolName:  aws.ToString(toolStart.Value.Name),
			}}
		}
	case *types.ConverseStreamOutputMemberContentBlockDelta:
		delta := v.Value.Delta
		switch d := delta.(type) {
		case *types.ContentBlockDeltaMemberText:
			if d.Value != "" {
				return []ir.StreamEvent{{
					Kind: ir.StreamTextDelta, ID: id, Model: model, Text: d.Value,
				}}
			}
		case *types.ContentBlockDeltaMemberToolUse:
			idx := 0
			if v.Value.ContentBlockIndex != nil {
				if mapped, ok := toolIndexes[int(*v.Value.ContentBlockIndex)]; ok {
					idx = mapped
				}
			}
			input := aws.ToString(d.Value.Input)
			if input != "" {
				return []ir.StreamEvent{{
					Kind: ir.StreamToolCallDelta, ID: id, Model: model,
					ToolIndex: idx, ArgsDelta: input,
				}}
			}
		}
	case *types.ConverseStreamOutputMemberMessageStop:
		finish := mapBedrockStopReason(v.Value.StopReason)
		return []ir.StreamEvent{{
			Kind: ir.StreamMessageEnd, ID: id, Model: model, FinishReason: finish,
		}}
	case *types.ConverseStreamOutputMemberMetadata:
		if v.Value.Usage != nil {
			usage := &ir.Usage{}
			if v.Value.Usage.InputTokens != nil {
				usage.PromptTokens = int64(*v.Value.Usage.InputTokens)
			}
			if v.Value.Usage.OutputTokens != nil {
				usage.CompletionTokens = int64(*v.Value.Usage.OutputTokens)
			}
			return []ir.StreamEvent{{
				Kind: ir.StreamMessageEnd, ID: id, Model: model, Usage: usage,
			}}
		}
	}
	return nil
}
