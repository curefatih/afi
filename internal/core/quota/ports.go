package quota

import (
	"context"

	"github.com/curefatih/afi/internal/core/auth"
	"github.com/curefatih/afi/internal/core/usage"
)

type Repository interface {
	List(
		ctx context.Context,
		scope Scope,
		targetID string,
	) ([]Quota, error)

	Commit(
		ctx context.Context,
		scope Scope,
		targetID string,
		usage []usage.Usage,
	) error
}

type PrincipalResolver interface {
	Scopes(auth.Principal) []Target
}
