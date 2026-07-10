package domain

import (
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type PartType string

const (
	PartTypeText     PartType = "text"
	PartTypeImageURL PartType = "image_url"
	PartTypeDocument PartType = "document"
)

type ImageDetail string

const (
	DetailAuto ImageDetail = "auto"
	DetailLow  ImageDetail = "low"
	DetailHigh ImageDetail = "high"
)

type TextPart struct {
	Text string `json:"text"`
}

type ImageURLPart struct {
	URL    string      `json:"url"`
	Detail ImageDetail `json:"detail,omitempty"`
}

type DocumentPart struct {
	Format    string `json:"format"` // e.g., "pdf"
	SourceURL string `json:"source_url"`
}

type ContentPart struct {
	Type     PartType      `json:"type"`
	Text     *TextPart     `json:"text,omitempty"`
	ImageURL *ImageURLPart `json:"image_url,omitempty"`
	Document *DocumentPart `json:"document,omitempty"`
}

type Message struct {
	Role       Role          `json:"role"`
	Parts      []ContentPart `json:"parts"`
	Name       string        `json:"name,omitempty"`         // Optional name for tool calls or specific users
	ToolCallID string        `json:"tool_call_id,omitempty"` // For associating tool responses
}

func NewTextMessage(role Role, text string) Message {
	return Message{
		Role: role,
		Parts: []ContentPart{
			{
				Type: PartTypeText,
				Text: &TextPart{Text: text},
			},
		},
	}
}

// RequestContext acts as the metadata envelope tracing the call through the multi-tenant hierarchy.
type RequestContext struct {
	OrganizationID string    `json:"organization_id"`
	TeamID         string    `json:"team_id"`
	ProjectID      string    `json:"project_id"`
	UserID         string    `json:"user_id,omitempty"` // Empty if authenticated purely via Service Account Key
	APIKeyHash     string    `json:"api_key_hash"`
	APIKeyType     string    `json:"api_key_type"` // PERSONAL or SERVICE_ACCOUNT
	TraceID        string    `json:"trace_id"`
	CallerIP       string    `json:"caller_ip"`
	ReceivedAt     time.Time `json:"received_at"`
}

// InternalRequest is the unified input payload for chat completion execution.
type InternalRequest struct {
	Messages      []Message      `json:"messages"`
	Model         string         `json:"model"` // Requested baseline model string (e.g., "gpt-4o")
	Temperature   float64        `json:"temperature"`
	TopP          float64        `json:"top_p"`
	MaxTokens     int            `json:"max_tokens"`
	Stream        bool           `json:"stream"`
	StopSequences []string       `json:"stop_sequences"`
	Metadata      RequestContext `json:"-"` // Kept outside standard structural payloads
}

type TokenUsage struct {
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	TotalTokens   int     `json:"total_tokens"`
	EstimatedCost float64 `json:"estimated_cost"` // Calculated based on localized model tier matrix
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // e.g., "stop", "length", "tool_calls"
}

// InternalResponse is the unified output payload for unary chat completion execution.
type InternalResponse struct {
	ID        string     `json:"id"`
	Object    string     `json:"object"` // typically "chat.completion"
	CreatedAt int64      `json:"created_at"`
	Model     string     `json:"model"` // The actual backend model that fulfilled the request
	Choices   []Choice   `json:"choices"`
	Usage     TokenUsage `json:"usage"`
}

type StreamChunk struct {
	ID        string     `json:"id"`
	CreatedAt int64      `json:"created_at"`
	Model     string     `json:"model"`
	DeltaText string     `json:"delta_text"`
	Usage     TokenUsage `json:"usage,omitempty"` // Usually populated only on the final chunk stream
}
