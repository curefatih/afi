package controlplane

import (
	"log/slog"

	"github.com/curefatih/afi/internal/app/platform"
)

// newPlatformEventBus builds the in-process bus with slog logging and optional durable outbox.
func newPlatformEventBus(log *slog.Logger, outbox platform.EventEnqueuer) *platform.Bus {
	bus := platform.NewBus()
	bus.SubscribeAll(platform.SlogHandler(log))
	if outbox != nil {
		bus.SubscribeAll(platform.OutboxHandler(outbox, log))
	}
	if log != nil {
		bus.OnPanic(func(recovered any, e platform.Event) {
			log.Error("platform event subscriber panic",
				"recovered", recovered,
				"event_id", e.ID,
				"name", string(e.Name),
			)
		})
	}
	return bus
}
