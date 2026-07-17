package pricing

import (
	"github.com/shopspring/decimal"

	"github.com/curefatih/afi/internal/core/usage"
)

type Price struct {
	Metric usage.Metric

	Unit decimal.Decimal
}
