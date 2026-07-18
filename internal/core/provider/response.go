package provider

import "github.com/curefatih/afi/internal/core/usage"

type Response struct {
	ID string

	Model string

	Content []Content

	ToolCalls []ToolCall

	FinishReason FinishReason

	Usage *usage.Report

	Metadata map[string]any
}
