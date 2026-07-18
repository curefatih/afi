package pricing

import (
	"context"

	"github.com/curefatih/afi/internal/core/usage"
)

type Service struct {
	repo Repository

	calculator Calculator
}

func NewService(
	repo Repository,
	calculator Calculator,
) *Service {
	return &Service{
		repo:       repo,
		calculator: calculator,
	}
}

func (s *Service) Calculate(
	ctx context.Context,
	modelID string,
	report *usage.Report,
) (*Money, error) {
	return s.calculator.Calculate(
		ctx,
		modelID,
		report,
	)
}
