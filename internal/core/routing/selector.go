package routing

import (
	"context"
)

type Selector interface {
	Select(
		ctx context.Context,
		routes []Route,
	) (*Decision, error)
}
