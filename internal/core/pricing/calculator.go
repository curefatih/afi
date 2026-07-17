package pricing

import (
	"github.com/curefatih/afi/internal/core/usage"
)

type Calculator interface {
	Calculate(
		pricing ModelPricing,
		report *usage.Report,
	) (*Money, error)
}
