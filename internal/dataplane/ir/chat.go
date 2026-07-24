// Package ir defines the gateway-owned chat internal representation.
package ir

import "github.com/curefatih/afi/sdk/chatir"

// Dialect identifies a client wire format.
type Dialect = chatir.Dialect

const (
	DialectOpenAI    = chatir.DialectOpenAI
	DialectAnthropic = chatir.DialectAnthropic
	DialectGemini    = chatir.DialectGemini
)

// ChatRequest is the dialect-neutral chat request.
type ChatRequest = chatir.Request

// Message is a single chat turn.
type Message = chatir.Message

// ContentPart is one piece of multimodal message content.
type ContentPart = chatir.ContentPart

// ContentPartType enumerates supported content part kinds.
type ContentPartType = chatir.ContentPartType

const (
	ContentText  = chatir.ContentText
	ContentImage = chatir.ContentImage
)

// ImageSource references image bytes either by URL or inline base64 data.
type ImageSource = chatir.ImageSource

// Tool is a function tool the model may call.
type Tool = chatir.Tool

// ToolChoiceMode controls whether/which tool the model must call.
type ToolChoiceMode = chatir.ToolChoiceMode

const (
	ToolChoiceAuto     = chatir.ToolChoiceAuto
	ToolChoiceNone     = chatir.ToolChoiceNone
	ToolChoiceRequired = chatir.ToolChoiceRequired
	ToolChoiceTool     = chatir.ToolChoiceTool
)

// ToolChoice expresses tool_choice across dialects.
type ToolChoice = chatir.ToolChoice

// ToolCall is a model-issued function call.
type ToolCall = chatir.ToolCall

// Usage holds token counts.
type Usage = chatir.Usage

// ChatResponse is a non-streaming assistant reply.
type ChatResponse = chatir.Response

// StreamEventKind classifies streaming IR events.
type StreamEventKind = chatir.StreamEventKind

const (
	StreamMessageStart  = chatir.StreamMessageStart
	StreamTextDelta     = chatir.StreamTextDelta
	StreamToolCallStart = chatir.StreamToolCallStart
	StreamToolCallDelta = chatir.StreamToolCallDelta
	StreamMessageEnd    = chatir.StreamMessageEnd
	StreamError         = chatir.StreamError
)

// StreamEvent is one IR streaming unit.
type StreamEvent = chatir.StreamEvent

// ChatResult is either a completed response, a stream of events, or an upstream HTTP error body.
type ChatResult = chatir.Result
