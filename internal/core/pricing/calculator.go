package pricing

import (
	"context"

	"github.com/curefatih/afi/internal/core/usage"
)

type Calculator interface {
	Calculate(
		ctx context.Context,
		modelID string,
		report *usage.Report,
	) (*Money, error)
}
