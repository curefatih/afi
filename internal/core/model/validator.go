package model

import (
	"github.com/curefatih/afi/internal/core/provider"
)

type Validator interface {
	Supports(
		model *Model,
		capability provider.Capability,
	) bool
}