package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/routing"
)

type ResolveRoute struct {
	service routing.Service
}

func NewResolveRoute(
	service routing.Service,
) *ResolveRoute {
	return &ResolveRoute{
		service: service,
	}
}

func (s *ResolveRoute) Name() string {
	return "resolve_route"
}

func (s *ResolveRoute) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	decision, err := s.service.Resolve(
		ctx,
		routing.ResolveRequest{
			Principal:  state.Principal(),
			ModelName:  state.Model().Name,
			Capability: state.Request().Payload.Capability(),
		},
	)

	if err != nil {
		return err
	}

	state.SetRoute(decision)

	state.Request().Provider.ID = decision.Target.ProviderID
	state.Request().Model.ProviderModelID = decision.Target.ProviderModelID

	return nil
}
