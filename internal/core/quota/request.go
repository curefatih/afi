package quota

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/pricing"
	"github.com/curefatih/afi/internal/core/usage"
)

type Request struct {
	Principal *auth.Principal

	Model *model.Model

	Usage *usage.Report

	Cost *pricing.Money
}
