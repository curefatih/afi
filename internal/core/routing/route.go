package routing

import "github.com/curefatih/afi/internal/core/provider"

type Route struct {
	ID          string
	Name        string
	Capability  provider.Capability
	Targets     []Target
	Strategy    Strategy
	Enabled     bool
	Metadata    map[string]string
	Description string
}
