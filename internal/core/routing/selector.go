package routing

import (
	"context"

	"github.com/curefatih/afi/internal/core/provider"
)

type Selector interface {
	Resolve(
		ctx context.Context,
		model string,
		capability provider.Capability,
	) (*Decision, error)
}