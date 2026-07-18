package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
)

type ExtractUsage struct{}

func NewExtractUsage() *ExtractUsage {
	return &ExtractUsage{}
}

func (s *ExtractUsage) Name() string {
	return "extract_usage"
}

func (s *ExtractUsage) Execute(
	_ context.Context,
	state *gateway.Context,
) error {

	if state.Response == nil {
		return nil
	}

	state.SetUsage(state.Response().Usage)

	return nil
}
