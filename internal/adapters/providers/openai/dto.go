package openai

import "github.com/curefatih/afi/internal/core/provider"

type ChatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    *float32        `json:"temperature"`
	TopP           *float32        `json:"top_p"`
	MaxTokens      *int            `json:"max_tokens"`
	Stop           []string        `json:"stop"`
	Stream         bool            `json:"stream"`
	ResponseFormat *ResponseFormat `json:"response_format"`
	Tools          []Tool          `json:"tools"`
}

type Tool struct {
	Type     string
	Function Function
}

type Function struct {
	Name        string
	Description string
	Parameters  any
}

type ResponseFormat struct {
	Type       string               `json:"type"`
	JSONSchema *provider.JSONSchema `json:"json_schema"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID string `json:"id"`

	Choices []Choice `json:"choices"`

	Usage Usage `json:"usage"`

	Model string `json:"model"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens int64 `json:"prompt_tokens"`

	CompletionTokens int64 `json:"completion_tokens"`

	TotalTokens int64 `json:"total_tokens"`
}
