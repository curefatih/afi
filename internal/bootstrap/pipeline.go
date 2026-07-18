package bootstrap

import (
	"github.com/curefatih/afi/internal/application/gateway"
	"github.com/curefatih/afi/internal/application/gateway/stages"
)

func buildPipeline(
	services *Services,
	providers *Providers,
) *gateway.Pipeline {

	return gateway.NewPipeline(

		stages.NewAuthenticate(
			services.Auth,
		),

		stages.NewResolveModel(
			services.Model,
		),

		stages.NewResolveRoute(
			services.Route,
		),

		stages.NewExecuteProvider(
			providers.Registry,
		),

		stages.NewExtractUsage(),

		stages.NewCalculatePricing(
			services.Pricing,
		),

		stages.NewCheckQuota(
			services.Quota,
		),

		stages.NewCommitUsage(
			services.Quota,
		),

		// stages.NewPublishEvents(
		// 	/* publisher */,
		// ),
	)
}
