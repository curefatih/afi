package stages

import (
	"context"

	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/core/model"
)

type ResolveModel struct {
	model model.Selector
}

func (s *ResolveModel) Name() string {
	return "ResolveModel"
}

func (s *ResolveModel) Execute(
	ctx context.Context,
	state *gateway.Context,
) error {

	model, err := s.model.Resolve(
		ctx,
		state.Request().Model.Name,
	)

	if err != nil {
		return err
	}

	state.SetModel(model)

	return nil
}
