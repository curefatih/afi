package event

import "context"

type Subscriber interface {
	Handle(context.Context, Event) error
}
