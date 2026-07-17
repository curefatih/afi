package provider

type EmbeddingRequest struct {
	Input []string

	Dimensions *int
}

func (EmbeddingRequest) Capability() Capability {
	return CapabilityEmbedding
}