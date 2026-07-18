package routing

import (
	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/provider"
)

type ResolveRequest struct {
	Principal *auth.Principal

	ModelName string

	Capability provider.Capability
}
