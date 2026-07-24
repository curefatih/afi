// Package ir defines the gateway-owned chat internal representation.
package ir

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
	Stream      bool
	Stop        []string
}

// Message is a single chat turn (v1: text content; tools/vision in backlog).
type Message struct {
	Role    string // system | user | assistant
	Content string
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
	FinishReason string // stop | length | …
	Usage        Usage
}

// StreamEventKind classifies streaming IR events.
type StreamEventKind string

const (
	StreamMessageStart StreamEventKind = "message_start"
	StreamTextDelta    StreamEventKind = "text_delta"
	StreamMessageEnd   StreamEventKind = "message_end"
	StreamError        StreamEventKind = "error"
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
