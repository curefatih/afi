package provider

type FinishReason string

const (
	FinishReasonStop FinishReason = "STOP"

	FinishReasonLength FinishReason = "LENGTH"

	FinishReasonToolCalls FinishReason = "TOOL_CALLS"

	FinishReasonContentFilter FinishReason = "CONTENT_FILTER"

	FinishReasonError FinishReason = "ERROR"
)