package routing

import (
	"context"
)

type Selector interface {
	Select(
		ctx context.Context,
		route Route,
	) (*Decision, error)
}
