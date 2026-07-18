package gateway

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/provider"
	"github.com/curefatih/afi/internal/core/usage"
)

type RequestCompleted struct {
	RequestID string

	Principal auth.Principal

	Provider string

	Model string

	Usage usage.Report

	Cost pricing.Money

	ResponseID string

	FinishReason provider.FinishReason
}
