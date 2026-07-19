package controlplane

import (
	"log/slog"

	"github.com/curefatih/afi/internal/app/platform"
)

// newPlatformEventBus builds the default in-process bus with slog debug logging.
func newPlatformEventBus(log *slog.Logger) *platform.Bus {
	bus := platform.NewBus()
	bus.SubscribeAll(platform.SlogHandler(log))
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
