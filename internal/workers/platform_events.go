package workers

import (
	"context"
	"encoding/json"

	"github.com/curefatih/afi/internal/app/platform"
)

// EventPublisher delivers a platform domain event to an external broker.
type EventPublisher interface {
	Publish(ctx context.Context, e platform.Event) error
}

// ProcessPlatformEventsOnce claims pending platform_event_outbox rows and publishes them.
func ProcessPlatformEventsOnce(ctx context.Context, src OutboxSource, pub EventPublisher) (int, error) {
	if pub == nil {
		return 0, nil
	}
	rows, err := src.ClaimBatch(ctx, 50)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, row := range rows {
		var e platform.Event
		if err := json.Unmarshal(row.Payload, &e); err != nil {
			return n, err
		}
		if err := pub.Publish(ctx, e); err != nil {
			return n, err
		}
		if err := src.MarkProcessed(ctx, row.ID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
