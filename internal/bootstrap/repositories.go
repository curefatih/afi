package bootstrap

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/quota"
	"github.com/curefatih/afi/internal/core/routing"
)

type Repositories struct {
	Auth auth.Repository

	Model model.Repository

	Route routing.Repository

	Quota quota.Repository

	Pricing pricing.Repository
}

func buildRepositories(
	cfg Config,
) (*Repositories, error) {

	return &Repositories{
		Auth:    newAuthRepository(cfg),
		Model:   newModelRepository(cfg),
		Route:   newRouteRepository(cfg),
		Quota:   newQuotaRepository(cfg),
		Pricing: newPricingRepository(cfg),
	}, nil
}
