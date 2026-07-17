package gateway

import (
	"context"

	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/routing"
	"github.com/curefatih/afi/internal/core/usage"
)

type Context struct {
	Context context.Context

	Request *Request

	Principal *auth.Principal

	Route *routing.Decision

	ProviderResponse *provider.Response

	Usage *usage.Report

	Cost *pricing.Money
}