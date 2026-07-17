package pricing

import "github.com/shopspring/decimal"

type Money struct {
	Currency string

	Amount decimal.Decimal
}
