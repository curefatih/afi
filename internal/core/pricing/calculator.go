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

type DefaultCalculator struct {
	priceBook *PriceBook
}

var _ Calculator = DefaultCalculator{}

func NewCalculator(priceBook *PriceBook) Calculator {
	return &DefaultCalculator{
		priceBook: priceBook,
	}

}

// Calculate implements Calculator.
func (d DefaultCalculator) Calculate(ctx context.Context, modelID string, report *usage.Report) (*Money, error) {
	panic("unimplemented")
}
