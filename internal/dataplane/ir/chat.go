// Package ir defines the gateway-owned chat internal representation.
package ir

import "encoding/json"

// Dialect identifies a client wire format.
type Dialect string

const (
	DialectOpenAI    Dialect = "openai"
	DialectAnthropic Dialect = "anthropic"
)

// ChatRequest is the dialect-neutral chat request.
type ChatRequest struct {
	Model       string
	Messages    []Message
	System      string
	MaxTokens   int
	Temperature *float64
	TopP        *float64
	Stream      bool
	Stop        []string
	Tools       []Tool
	ToolChoice  *ToolChoice
}

// Message is a single chat turn.
//
// Content holds plain text and is always populated for convenience. Parts holds
// multimodal content (text + images) and, when non-empty, is the authoritative
// representation. ToolCalls carry assistant-issued function calls; ToolCallID is
// set on role=="tool" messages to reference the call being answered.
type Message struct {
	Role       string // system | user | assistant | tool
	Content    string
	Parts      []ContentPart
	ToolCalls  []ToolCall
	ToolCallID string
}

// ContentPart is one piece of multimodal message content.
type ContentPart struct {
	Type  ContentPartType
	Text  string       // Type == ContentText
	Image *ImageSource // Type == ContentImage
}

// ContentPartType enumerates supported content part kinds.
type ContentPartType string

const (
	ContentText  ContentPartType = "text"
	ContentImage ContentPartType = "image"
)

// ImageSource references image bytes either by URL or inline base64 data.
type ImageSource struct {
	URL       string // http(s) URL; empty when inline data is used
	MediaType string // e.g. "image/png"; required for inline data
	Data      string // base64-encoded bytes (no data: prefix); empty when URL is used
}

// Tool is a function tool the model may call.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage // JSON Schema object
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
	Mode ToolChoiceMode
	Name string // set when Mode == ToolChoiceTool
}

// ToolCall is a model-issued function call.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON-encoded arguments object
}

// Usage holds token counts.
type Usage struct {
	PromptTokens     int64
	CompletionTokens int64
}

// ChatResponse is a non-streaming assistant reply.
type ChatResponse struct {
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

// ChatResult is either a completed response, a stream of events, or an upstream HTTP error body.
type ChatResult struct {
	StatusCode int
	Header     map[string][]string
	Response   *ChatResponse
	// Events is closed when the stream ends. Nil for non-stream success.
	Events <-chan StreamEvent
	// ErrorBody is set when StatusCode >= 400 and the upstream body was not mapped to IR.
	ErrorBody []byte
}
