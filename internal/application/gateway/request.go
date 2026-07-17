package gateway

import (
	"github.com/curefatih/afi/internal/core/model"
	"github.com/curefatih/afi/internal/core/provider"
)

type Request struct {
	APIKey string

	Model *model.Model

	Capability provider.Capability

	Body any

	Metadata map[string]any
}
