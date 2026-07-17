package routing

import (
	"context"

	"github.com/curefatih/afi/internal/core/provider"
)

type Repository interface {
	FindByCapability(
		ctx context.Context,
		capability provider.Capability,
	) (*Route, error)

	FindByName(
		ctx context.Context,
		name string,
	) (*Route, error)
}

type HealthChecker interface {
	IsHealthy(providerID string) bool
}
