package quota

import (
	"context"

	"github.com/curefatih/afi/internal/core/auth"
)

type Repository interface {
	List(
		ctx context.Context,
		scope Scope,
		targetID string,
	) ([]Quota, error)

	AddUsage(
		ctx context.Context,
		scope Scope,
		targetID string,
		usage []Usage,
	) error
}

type PrincipalResolver interface {
	Scopes(auth.Principal) []Target
}
