package provider

type Response struct {
	ID string

	Model string

	Content []Content

	ToolCalls []ToolCall

	FinishReason FinishReason

	Usage Usage

	Metadata map[string]any
}
