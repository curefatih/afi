package workers

import (
	"context"
	"encoding/json"

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// EventPublisher delivers a platform domain event to an external broker.
type EventPublisher interface {
	Publish(ctx context.Context, e platform.Event) error
}

// ProcessPlatformEventsOnce claims pending platform_event_outbox rows and publishes them.
func ProcessPlatformEventsOnce(ctx context.Context, src OutboxSource, pub EventPublisher, metrics ...*telemetry.WorkerMetrics) (int, error) {
	if pub == nil {
		return 0, nil
	}
	var m *telemetry.WorkerMetrics
	if len(metrics) > 0 {
		m = metrics[0]
	}
	ctx, span := telemetry.Tracer("afi.worker").Start(ctx, "afi.worker.process_platform_events")
	defer span.End()

	rows, err := src.ClaimBatch(ctx, 50)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	if m != nil && n > 0 {
		m.PlatformEventsPublished.Add(ctx, int64(n))
	}
	span.SetAttributes(attribute.Int("afi.worker.batch_size", n))
	return n, nil
}
