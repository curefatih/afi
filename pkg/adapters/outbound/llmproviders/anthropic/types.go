package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/curefatih/afi/internal/core/domain"
)

type AnthropicPart struct {
	Type   string `json:"type"` // "text" or "image"
	Text   string `json:"text,omitempty"`
	Source *struct {
		Type      string `json:"type"`       // "base64"
		MediaType string `json:"media_type"` // "image/jpeg", "image/png" etc
		Data      string `json:"data"`       // raw base64 encoded string
	} `json:"source,omitempty"`
}

type AnthropicMessage struct {
	Role    string          `json:"role"` // Strictly "user" or "assistant"
	Content []AnthropicPart `json:"content"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"` // Global prompt extraction goes here
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	Stream      bool               `json:"stream,omitempty"`
}

type AnthropicResponse struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	Role  string `json:"role"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// mapToAnthropic converts an internal request representation into standard unary Anthropic layouts.
func mapToAnthropic(req *domain.InternalRequest) any {
	var systemPrompt string
	var anthropicMessages []AnthropicMessage

	for _, msg := range req.Messages {
		// Anthropic extracts 'system' roles out into a dedicated global configuration block
		if msg.Role == domain.RoleSystem {
			if len(msg.Parts) > 0 && msg.Parts[0].Text != nil {
				systemPrompt = msg.Parts[0].Text.Text
			}
			continue
		}

		parts := make([]AnthropicPart, 0, len(msg.Parts))
		for _, p := range msg.Parts {
			switch p.Type {
			case domain.PartTypeText:
				if p.Text != nil {
					parts = append(parts, AnthropicPart{
						Type: "text",
						Text: p.Text.Text,
					})
				}
			case domain.PartTypeImageURL:
				// If parsing base64 data strings directly (data:image/png;base64,...)
				if p.ImageURL != nil && isBase64DataURI(p.ImageURL.URL) {
					mediaType, base64Data := parseBase64URI(p.ImageURL.URL)
					parts = append(parts, AnthropicPart{
						Type: "image",
						Source: &struct {
							Type      string `json:"type"`
							MediaType string `json:"media_type"`
							Data      string `json:"data"`
						}{
							Type:      "base64",
							MediaType: mediaType,
							Data:      base64Data,
						},
					})
				}
			}
		}

		anthropicMessages = append(anthropicMessages, AnthropicMessage{
			Role:    string(msg.Role),
			Content: parts,
		})
	}

	// Default fallback cap protection required by Anthropic's strict validation
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	return &AnthropicRequest{
		Model:       req.Model,
		System:      systemPrompt,
		Messages:    anthropicMessages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
}

// mapToAnthropicStreamReq maps structural data identical to unary operations with Stream validation activated.
func mapToAnthropicStreamReq(req *domain.InternalRequest) any {
	anthropicReq := mapToAnthropic(req).(*AnthropicRequest)
	anthropicReq.Stream = true
	return anthropicReq
}

// mapToInternalResponse extracts response signatures seamlessly into standardized structures.
func mapToInternalResponse(resp *http.Response) (*domain.InternalResponse, error) {
	var vendorResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&vendorResp); err != nil {
		return nil, fmt.Errorf("failed to extract anthropic structure representation: %w", err)
	}

	var generatedText string
	if len(vendorResp.Content) > 0 {
		generatedText = vendorResp.Content[0].Text
	}

	choices := []domain.Choice{
		{
			Index:        0,
			FinishReason: "stop",
			Message:      domain.NewTextMessage(domain.RoleAssistant, generatedText),
		},
	}

	return &domain.InternalResponse{
		ID:        vendorResp.ID,
		Object:    "chat.completion",
		CreatedAt: time.Now().Unix(),
		Model:     vendorResp.Model,
		Choices:   choices,
		Usage: domain.TokenUsage{
			InputTokens:  vendorResp.Usage.InputTokens,
			OutputTokens: vendorResp.Usage.OutputTokens,
			TotalTokens:  vendorResp.Usage.InputTokens + vendorResp.Usage.OutputTokens,
		},
	}, nil
}

func isBase64DataURI(url string) bool {
	return len(url) > 5 && url[:5] == "data:"
}

func parseBase64URI(url string) (mediaType string, data string) {
	// Simple string dissection helper for "data:image/png;base64,iVBORw0KGgo..."
	commaIdx := -1
	for i := 0; i < len(url); i++ {
		if url[i] == ',' {
			commaIdx = i
			break
		}
	}
	if commaIdx == -1 {
		return "image/jpeg", url
	}

	header := url[:commaIdx]
	data = url[commaIdx+1:]

	mediaType = "image/jpeg"
	if len(header) > 5 {
		subSection := header[5:]
		for i := 0; i < len(subSection); i++ {
			if subSection[i] == ';' {
				mediaType = subSection[:i]
				break
			}
		}
	}
	return mediaType, data
}
