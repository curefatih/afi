package gemini

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/curefatih/afi/internal/core/domain"
)

// Gemini Schema representations
type GeminiPart struct {
	Text       string          `json:"text,omitempty"`
	InlineData *GeminiBlobData `json:"inlineData,omitempty"`
}

type GeminiBlobData struct {
	MimeType string `json:"mimeType"` // e.g. "image/png"
	Data     string `json:"data"`     // base64 encoded payload
}

type GeminiContent struct {
	Role  string       `json:"role"` // Strictly "user" or "model"
	Parts []GeminiPart `json:"parts"`
}

type SystemInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type GeminiRequest struct {
	Contents          []GeminiContent    `json:"contents"`
	SystemInstruction *SystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig  `json:"generationConfig,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// mapToGemini translates internal structures to Gemini JSON structures
func mapToGemini(req *domain.InternalRequest) *GeminiRequest {
	var systemInstruction *SystemInstruction
	var geminiContents []GeminiContent

	for _, msg := range req.Messages {
		// Extract system messages to system instruction blocks
		if msg.Role == domain.RoleSystem {
			if len(msg.Parts) > 0 && msg.Parts[0].Text != nil {
				systemInstruction = &SystemInstruction{
					Parts: []GeminiPart{{Text: msg.Parts[0].Text.Text}},
				}
			}
			continue
		}

		parts := make([]GeminiPart, 0, len(msg.Parts))
		for _, p := range msg.Parts {
			switch p.Type {
			case domain.PartTypeText:
				if p.Text != nil {
					parts = append(parts, GeminiPart{Text: p.Text.Text})
				}
			case domain.PartTypeImageURL:
				if p.ImageURL != nil && isBase64DataURI(p.ImageURL.URL) {
					mimeType, base64Data := parseBase64URI(p.ImageURL.URL)
					parts = append(parts, GeminiPart{
						InlineData: &GeminiBlobData{
							MimeType: mimeType,
							Data:     base64Data,
						},
					})
				}
			}
		}

		// map assistant -> model
		role := string(msg.Role)
		if role == "assistant" {
			role = "model"
		}

		geminiContents = append(geminiContents, GeminiContent{
			Role:  role,
			Parts: parts,
		})
	}

	config := &GenerationConfig{
		Temperature:     req.Temperature,
		MaxOutputTokens: req.MaxTokens,
	}

	return &GeminiRequest{
		Contents:          geminiContents,
		SystemInstruction: systemInstruction,
		GenerationConfig:  config,
	}
}

func mapToInternalResponse(resp *http.Response, model string) (*domain.InternalResponse, error) {
	var vendorResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&vendorResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	var generatedText string
	finishReason := "stop"

	if len(vendorResp.Candidates) > 0 {
		candidate := vendorResp.Candidates[0]
		if len(candidate.Content.Parts) > 0 {
			generatedText = candidate.Content.Parts[0].Text
		}
		if candidate.FinishReason != "" {
			finishReason = candidate.FinishReason
		}
	}

	choices := []domain.Choice{
		{
			Index:        0,
			FinishReason: finishReason,
			Message:      domain.NewTextMessage(domain.RoleAssistant, generatedText),
		},
	}

	return &domain.InternalResponse{
		ID:        fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Object:    "chat.completion",
		CreatedAt: time.Now().Unix(),
		Model:     model,
		Choices:   choices,
		Usage: domain.TokenUsage{
			InputTokens:  vendorResp.UsageMetadata.PromptTokenCount,
			OutputTokens: vendorResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  vendorResp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func isBase64DataURI(url string) bool {
	return len(url) > 5 && url[:5] == "data:"
}

func parseBase64URI(url string) (mimeType string, data string) {
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

	mimeType = "image/jpeg"
	if len(header) > 5 {
		subSection := header[5:]
		for i := 0; i < len(subSection); i++ {
			if subSection[i] == ';' {
				mimeType = subSection[:i]
				break
			}
		}
	}
	return mimeType, data
}
