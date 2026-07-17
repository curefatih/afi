package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/pricing"
)

type CalculatePricing struct {
	service pricing.Calculator
}

func NewCalculatePricing(
	service pricing.Calculator,
) *CalculatePricing {
	return &CalculatePricing{
		service: service,
	}
}

func (s *CalculatePricing) Name() string {
	return "calculate_pricing"
}

func (s *CalculatePricing) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	if state.Usage == nil {
		return nil
	}

	money, err := s.service.Calculate(
		ctx,
		state.Model().ID,
		state.Usage(),
	)

	if err != nil {
		return err
	}

	state.SetCost(money)

	return nil
}
