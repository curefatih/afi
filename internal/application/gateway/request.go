package gateway

import (
	"github.com/curefatih/afi/internal/core/provider"
)

type Request struct {
	APIKey string

	Model string

	Capability provider.Capability

	Body any

	Metadata map[string]any
}
