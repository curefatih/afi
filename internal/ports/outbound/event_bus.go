package outbound

import (
	"context"

	"github.com/curefatih/afi/internal/core/event"
)

type EventBus interface {
	Publish(
		ctx context.Context,
		events ...event.Event,
	) error
}
