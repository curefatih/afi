package openai

import "github.com/curefatih/afi/internal/core/domain"

type OpenAIRequestPayload struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	Stream      bool            `json:"stream"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (h *OpenAIHandler) mapToDomainRequest(src *OpenAIRequestPayload, ctx *domain.RequestContext) *domain.InternalRequest {
	domainMessages := make([]domain.Message, len(src.Messages))
	for i, m := range src.Messages {
		domainMessages[i] = domain.NewTextMessage(domain.Role(m.Role), m.Content)
	}

	return &domain.InternalRequest{
		Model:       src.Model,
		Messages:    domainMessages,
		Temperature: src.Temperature,
		Stream:      src.Stream,
		Metadata:    *ctx,
	}
}

// Target outbound contract representations reflecting matching corporate shapes
type OpenAIResponseLayout struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func mapFromDomainResponse(src *domain.InternalResponse) *OpenAIResponseLayout {
	dest := &OpenAIResponseLayout{
		ID:      src.ID,
		Object:  "chat.completion",
		Created: src.CreatedAt,
		Model:   src.Model,
	}
	for _, c := range src.Choices {
		var txt string
		if len(c.Message.Parts) > 0 && c.Message.Parts[0].Text != nil {
			txt = c.Message.Parts[0].Text.Text
		}
		dest.Choices = append(dest.Choices, struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			Index: c.Index,
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{Role: string(c.Message.Role), Content: txt},
			FinishReason: c.FinishReason,
		})
	}
	return dest
}

func mapFromDomainChunk(src *domain.StreamChunk) any {
	// Truncated mockup structural definition mimicking common chunk outputs
	return map[string]any{
		"id":      src.ID,
		"object":  "chat.completion.chunk",
		"created": src.CreatedAt,
		"model":   src.Model,
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{"content": src.DeltaText},
			},
		},
	}
}
