package bootstrap

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/quota"
	"github.com/curefatih/afi/internal/core/routing"
)

type Services struct {
	Auth *auth.Service

	Model *model.Service

	Route *routing.Service

	Quota *quota.Service

	Pricing *pricing.Service
}

func buildServices(
	repos *Repositories,
	priceBook *pricing.PriceBook,
	modelRegistry model.Registry,
	modelValidator model.Validator,
	modelSelector model.Selector,
	routingSelector routing.Selector,
	principalResolver quota.PrincipalResolver,
) (*Services, error) {

	return &Services{
		Auth: auth.NewService(
			repos.Auth,
			auth.NewKeyGenerator(),
			auth.NewSHA256Hasher(),
		),

		Model: model.NewService(
			repos.Model,
			modelRegistry,
			modelValidator,
			modelSelector,
		),

		Route: routing.NewService(
			repos.Route,
			routingSelector,
		),

		Quota: quota.NewService(
			repos.Quota,
			principalResolver,
			quota.Evaluator{},
		),

		Pricing: pricing.NewService(
			repos.Pricing,
			pricing.NewCalculator(priceBook),
		),
	}, nil
}
