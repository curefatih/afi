// Package chatir defines the gateway-owned chat internal representation used by
// in-process SDK adapters and (via protobuf) gRPC extensions.
package chatir

import "encoding/json"

// Dialect identifies a client wire format.
type Dialect string

const (
	DialectOpenAI    Dialect = "openai"
	DialectAnthropic Dialect = "anthropic"
	DialectGemini    Dialect = "gemini"
)

// Request is the dialect-neutral chat request.
type Request struct {
	Model       string      `json:"model"`
	Messages    []Message   `json:"messages"`
	System      string      `json:"system,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
	Temperature *float64    `json:"temperature,omitempty"`
	TopP        *float64    `json:"top_p,omitempty"`
	Stream      bool        `json:"stream,omitempty"`
	Stop        []string    `json:"stop,omitempty"`
	Tools       []Tool      `json:"tools,omitempty"`
	ToolChoice  *ToolChoice `json:"tool_choice,omitempty"`
}

// Message is a single chat turn.
//
// Content holds plain text and is always populated for convenience. Parts holds
// multimodal content (text + images) and, when non-empty, is the authoritative
// representation. ToolCalls carry assistant-issued function calls; ToolCallID is
// set on role=="tool" messages to reference the call being answered.
type Message struct {
	Role       string        `json:"role"` // system | user | assistant | tool
	Content    string        `json:"content,omitempty"`
	Parts      []ContentPart `json:"parts,omitempty"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

// ContentPart is one piece of multimodal message content.
type ContentPart struct {
	Type  ContentPartType `json:"type"`
	Text  string          `json:"text,omitempty"`  // Type == ContentText
	Image *ImageSource    `json:"image,omitempty"` // Type == ContentImage
}

// ContentPartType enumerates supported content part kinds.
type ContentPartType string

const (
	ContentText  ContentPartType = "text"
	ContentImage ContentPartType = "image"
)

// ImageSource references image bytes either by URL or inline base64 data.
type ImageSource struct {
	URL       string `json:"url,omitempty"`        // http(s) URL; empty when inline data is used
	MediaType string `json:"media_type,omitempty"` // e.g. "image/png"; required for inline data
	Data      string `json:"data,omitempty"`       // base64-encoded bytes (no data: prefix); empty when URL is used
}

// Tool is a function tool the model may call.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema object
}

// ToolChoiceMode controls whether/which tool the model must call.
type ToolChoiceMode string

const (
	ToolChoiceAuto     ToolChoiceMode = "auto"
	ToolChoiceNone     ToolChoiceMode = "none"
	ToolChoiceRequired ToolChoiceMode = "required" // any tool
	ToolChoiceTool     ToolChoiceMode = "tool"     // a specific named tool
)

// ToolChoice expresses tool_choice across dialects.
type ToolChoice struct {
	Mode ToolChoiceMode `json:"mode"`
	Name string         `json:"name,omitempty"` // set when Mode == ToolChoiceTool
}

// ToolCall is a model-issued function call.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded arguments object
}

// Usage holds token counts.
type Usage struct {
	PromptTokens     int64
	CompletionTokens int64
}

// Response is a non-streaming assistant reply.
type Response struct {
	ID           string
	Model        string
	Role         string
	Content      string
	ToolCalls    []ToolCall
	FinishReason string // stop | length | tool_calls | …
	Usage        Usage
}

// StreamEventKind classifies streaming IR events.
type StreamEventKind string

const (
	StreamMessageStart  StreamEventKind = "message_start"
	StreamTextDelta     StreamEventKind = "text_delta"
	StreamToolCallStart StreamEventKind = "tool_call_start"
	StreamToolCallDelta StreamEventKind = "tool_call_delta"
	StreamMessageEnd    StreamEventKind = "message_end"
	StreamError         StreamEventKind = "error"
)

// StreamEvent is one IR streaming unit.
type StreamEvent struct {
	Kind         StreamEventKind
	ID           string
	Model        string
	Role         string
	Text         string
	FinishReason string
	Usage        *Usage
	Err          error

	// Tool-call streaming fields (StreamToolCallStart / StreamToolCallDelta).
	ToolIndex int
	ToolID    string
	ToolName  string
	ArgsDelta string
}

// Result is either a completed response, a stream of events, or an upstream HTTP error body.
type Result struct {
	StatusCode int
	Header     map[string][]string
	Response   *Response
	// Events is closed when the stream ends. Nil for non-stream success.
	Events <-chan StreamEvent
	// ErrorBody is set when StatusCode >= 400 and the upstream body was not mapped to IR.
	ErrorBody []byte
}
