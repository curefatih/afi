package quota

type Metric string

const (
	MetricInputTokens  Metric = "INPUT_TOKENS"
	MetricOutputTokens Metric = "OUTPUT_TOKENS"
	MetricCachedTokens Metric = "CACHED_TOKENS"

	MetricImages Metric = "IMAGES"

	MetricAudioSeconds Metric = "AUDIO_SECONDS"

	MetricEmbeddingTokens Metric = "EMBEDDING_TOKENS"

	MetricRequests Metric = "REQUESTS"

	MetricCost Metric = "COST"
)
