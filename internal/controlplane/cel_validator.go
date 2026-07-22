package controlplane

import "github.com/curefatih/afi/internal/policy"

// celValidator adapts policy.Validate / ValidateString to gatewayconfig.ExpressionValidator.
type celValidator struct{}

func (celValidator) Validate(expression string) error {
	return policy.Validate(expression)
}

func (celValidator) ValidateString(expression string) error {
	return policy.ValidateString(expression)
}
