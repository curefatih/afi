package openai

import (
	"fmt"

	"github.com/curefatih/afi/internal/core/provider"
)

type Translator struct{}

func NewTranslator() *Translator {
	return &Translator{}
}

func (t *Translator) Encode(req *provider.Request) (*ChatCompletionRequest, error) {
	payload, ok := req.Payload.(provider.ChatRequest)
	if !ok {
		if p, ok := req.Payload.(*provider.ChatRequest); ok {
			payload = *p
		} else {
			return nil, fmt.Errorf("openai translator: expected ChatRequest payload")
		}
	}

	dto := &ChatCompletionRequest{
		Model: req.Model.ProviderID,
	}

	// System prompt
	if payload.System != "" {
		dto.Messages = append(dto.Messages, Message{
			Role:    "system",
			Content: payload.System,
		})
	}

	// Conversation
	for _, msg := range payload.Messages {
		m := Message{
			Role: string(msg.Role),
		}

		// OpenAI Chat Completions supports string content.
		// We'll concatenate text parts and ignore unsupported
		// content types until Responses API is implemented.
		for _, part := range msg.Content {
			switch c := part.(type) {

			case provider.TextContent:
				if m.Content != "" {
					m.Content += "\n"
				}
				m.Content += c.Text

			case *provider.TextContent:
				if m.Content != "" {
					m.Content += "\n"
				}
				m.Content += c.Text

			default:
				return nil, fmt.Errorf(
					"unsupported content type %T for chat completions",
					part,
				)
			}
		}

		dto.Messages = append(dto.Messages, m)
	}

	// Sampling
	dto.Temperature = payload.Temperature
	dto.TopP = payload.TopP
	dto.MaxTokens = payload.MaxTokens
	dto.Stop = payload.Stop
	dto.Stream = payload.Stream

	// Response format
	if payload.ResponseFormat != nil {
		switch payload.ResponseFormat.Type {

		case provider.ResponseFormatJSON:
			dto.ResponseFormat = &ResponseFormat{
				Type: "json_object",
			}

		case provider.ResponseFormatJSONSchema:
			dto.ResponseFormat = &ResponseFormat{
				Type:       "json_schema",
				JSONSchema: payload.ResponseFormat.Schema,
			}
		}
	}

	// Tools
	for _, tool := range payload.Tools {
		dto.Tools = append(dto.Tools, Tool{
			Type: "function",
			Function: Function{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	return dto, nil
}

func (t *Translator) Decode(resp *ChatCompletionResponse) (*provider.Response, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai response contains no choices")
	}

	choice := resp.Choices[0]

	result := &provider.Response{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: provider.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
		Metadata: map[string]any{},
	}

	// Assistant text
	if choice.Message.Content != "" {
		result.Content = []provider.Content{
			provider.TextContent{
				Text: choice.Message.Content,
			},
		}
	}

	// // Tool calls
	// for _, tc := range choice.Message.ToolCalls {
	// 	result.ToolCalls = append(result.ToolCalls, provider.ToolCall{
	// 		ID:        tc.ID,
	// 		Name:      tc.Function.Name,
	// 		Arguments: tc.Function.Arguments,
	// 	})
	// }

	// Finish reason
	switch choice.FinishReason {

	case "stop":
		result.FinishReason = provider.FinishReasonStop

	case "length":
		result.FinishReason = provider.FinishReasonLength

	case "tool_calls":
		result.FinishReason = provider.FinishReasonToolCalls

	case "content_filter":
		result.FinishReason = provider.FinishReasonContentFilter

	default:
		result.FinishReason = provider.FinishReasonError
	}

	return result, nil
}
