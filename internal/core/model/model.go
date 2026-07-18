package model

import (
	"github.com/curefatih/afi/internal/core/provider"
)

type Model struct {
	ID string

	ProviderID string

	Name string

	Description string

	Capabilities []provider.Capability

	ContextWindow int

	MaxOutputTokens int

	Enabled bool
}
