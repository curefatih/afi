package provider

type Usage struct {
	InputTokens int64

	OutputTokens int64

	ReasoningTokens int64

	CacheReadTokens int64

	CacheWriteTokens int64

	ImageCount int64

	AudioSeconds float64
}