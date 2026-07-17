package event

import "context"

type Dispatcher interface {
	Dispatch(
		context.Context,
		Event,
	) error
}
