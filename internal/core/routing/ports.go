package routing

import (
	"context"
)

type Repository interface {
	FindCandidates(
		ctx context.Context,
		request ResolveRequest,
	) ([]Route, error)

	Find(
		ctx context.Context,
		request ResolveRequest,
	) (*Route, error)
}

type HealthChecker interface {
	IsHealthy(providerID string) bool
}
