package openai

import "github.com/curefatih/afi/internal/core/domain"

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"` // Simplified text serialization for now
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type OpenAIRequest struct {
	Model         string          `json:"model"`
	Messages      []OpenAIMessage `json:"messages"`
	Temperature   float64         `json:"temperature"`
	MaxTokens     int             `json:"max_tokens,omitempty"`
	Stream        bool            `json:"stream"`
	StreamOptions *StreamOptions  `json:"stream_options,omitempty"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type VendorStreamChunk struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// =========================================================================
// Mapping Translators
// =========================================================================

func mapToOpenAIRequest(req *domain.InternalRequest) *OpenAIRequest {
	messages := make([]OpenAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		var contentStr string
		if len(m.Parts) > 0 && m.Parts[0].Text != nil {
			contentStr = m.Parts[0].Text.Text
		}
		messages[i] = OpenAIMessage{
			Role:    string(m.Role),
			Content: contentStr,
		}
	}
	return &OpenAIRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
}

func mapToInternalResponse(v *OpenAIResponse) *domain.InternalResponse {
	choices := make([]domain.Choice, len(v.Choices))
	for i, c := range v.Choices {
		choices[i] = domain.Choice{
			Index:        i,
			FinishReason: c.FinishReason,
			Message:      domain.NewTextMessage(domain.Role(c.Message.Role), c.Message.Content),
		}
	}
	return &domain.InternalResponse{
		ID:        v.ID,
		CreatedAt: v.Created,
		Model:     v.Model,
		Choices:   choices,
		Usage: domain.TokenUsage{
			InputTokens:  v.Usage.PromptTokens,
			OutputTokens: v.Usage.CompletionTokens,
			TotalTokens:  v.Usage.TotalTokens,
		},
	}
}
