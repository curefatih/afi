package platform

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// EventEnqueuer persists platform events for durable cross-process delivery.
type EventEnqueuer interface {
	Enqueue(ctx context.Context, payload []byte) error
}

// OutboxHandler returns a bus Handler that enqueues events to a durable outbox.
// Enqueue failures are logged and do not fail the request path.
func OutboxHandler(enq EventEnqueuer, log *slog.Logger) Handler {
	return func(ctx context.Context, e Event) {
		if enq == nil {
			return
		}
		if e.ID == "" {
			e.ID = newEventID()
		}
		if e.At.IsZero() {
			e.At = time.Now().UTC()
		}
		payload, err := json.Marshal(e)
		if err != nil {
			if log != nil {
				log.Error("encode platform event for outbox", "err", err, "name", string(e.Name))
			}
			return
		}
		if err := enq.Enqueue(ctx, payload); err != nil && log != nil {
			log.Error("enqueue platform event", "err", err, "name", string(e.Name), "event_id", e.ID)
		}
	}
}
