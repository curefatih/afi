package gateway

import (
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/core/usage"
)

type Response struct {
	Response *provider.Response

	Usage *usage.Report

	Cost *pricing.Money
}