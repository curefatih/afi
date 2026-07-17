package pricing

import "context"

type Repository interface {
	FindModelPricing(
		ctx context.Context,
		providerID string,
		modelID string,
	) (*ModelPricing, error)
}
