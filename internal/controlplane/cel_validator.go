package controlplane

import "github.com/curefatih/afi/internal/policy"

// celValidator adapts policy.Validate to gatewayconfig.ExpressionValidator.
type celValidator struct{}

func (celValidator) Validate(expression string) error {
	return policy.Validate(expression)
}
