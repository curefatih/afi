package usage

type Metric string

const (
	// Text
	MetricInputTokens  Metric = "INPUT_TOKENS"
	MetricOutputTokens Metric = "OUTPUT_TOKENS"
	MetricCachedTokens Metric = "CACHED_TOKENS"

	// Embeddings
	MetricEmbeddingTokens Metric = "EMBEDDING_TOKENS"

	// Images
	MetricImagesGenerated Metric = "IMAGES_GENERATED"

	// Audio
	MetricAudioInputSeconds  Metric = "AUDIO_INPUT_SECONDS"
	MetricAudioOutputSeconds Metric = "AUDIO_OUTPUT_SECONDS"

	// Video
	MetricVideoSeconds Metric = "VIDEO_SECONDS"

	// Generic
	MetricRequests Metric = "REQUESTS"

	// Pricing can also emit this
	MetricCost Metric = "COST"
)
