package gateway

import "context"

type Stage interface {
	Name() string

	Execute(
		context.Context,
		*Context,
	) error
}
