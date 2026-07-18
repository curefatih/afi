package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/provider"
)

type ExecuteProvider struct {
	registry provider.Registry
}

func NewExecuteProvider(registry provider.Registry) *ExecuteProvider {
	return &ExecuteProvider{
		registry: registry,
	}
}

func (s *ExecuteProvider) Name() string {
	return "ExecuteProvider"
}

func (s *ExecuteProvider) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	client, err := s.registry.Client(
		state.Route().Route.ID,
	)

	resp, err := client.Execute(
		ctx,
		state.Request(),
	)

	if err != nil {
		return err
	}

	state.SetResponse(resp)

	return nil
}
