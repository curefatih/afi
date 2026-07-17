package usage

import "github.com/shopspring/decimal"

type Usage struct {
	Metric Metric

	Value decimal.Decimal
}
