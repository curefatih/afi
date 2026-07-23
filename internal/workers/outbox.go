package workers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/curefatih/afi/internal/telemetry"
	"github.com/curefatih/afi/internal/usage"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type OutboxRow struct {
	ID          int64
	Payload     []byte
	ProcessedAt *time.Time
}

// UsagePayload is an alias for the canonical usage.Event carried in the outbox.
type UsagePayload = usage.Event

// UsageEnqueuer is the gateway write-side port for usage_outbox.
type UsageEnqueuer interface {
	Enqueue(ctx context.Context, payload []byte) error
}

type OutboxSource interface {
	ClaimBatch(ctx context.Context, limit int) ([]OutboxRow, error)
	MarkProcessed(ctx context.Context, id int64) error
}

type PriceLookup interface {
	LookupModelPrice(ctx context.Context, providerType, model string) (inputPerMTok, outputPerMTok float64, ok bool, err error)
}

type UsageSink interface {
	InsertUsage(ctx context.Context, e usage.Event, costUSD *float64) error
}

// ProcessOnce claims pending outbox rows and writes usage_events.
// Optional metrics argument records worker counters when non-nil.
func ProcessOnce(ctx context.Context, src OutboxSource, sink UsageSink, prices PriceLookup, metrics ...*telemetry.WorkerMetrics) (int, error) {
	var m *telemetry.WorkerMetrics
	if len(metrics) > 0 {
		m = metrics[0]
	}
	start := time.Now()
	ctx, span := telemetry.Tracer("afi.worker").Start(ctx, "afi.worker.process_usage")
	defer span.End()

	rows, err := src.ClaimBatch(ctx, 50)
	if err != nil {
		if m != nil {
			m.UsageErrors.Add(ctx, 1)
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	n := 0
	for _, row := range rows {
		var payload UsagePayload
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			if m != nil {
				m.UsageErrors.Add(ctx, 1)
			}
			return n, err
		}
		cost, err := estimateUsageCost(ctx, payload, prices)
		if err != nil {
			if m != nil {
				m.UsageErrors.Add(ctx, 1)
			}
			return n, err
		}
		if err := sink.InsertUsage(ctx, payload, cost); err != nil {
			if m != nil {
				m.UsageErrors.Add(ctx, 1)
			}
			return n, err
		}
		if err := src.MarkProcessed(ctx, row.ID); err != nil {
			if m != nil {
				m.UsageErrors.Add(ctx, 1)
			}
			return n, err
		}
		n++
	}
	if m != nil {
		if n > 0 {
			m.UsageProcessed.Add(ctx, int64(n))
		}
		m.ProcessDuration.Record(ctx, time.Since(start).Seconds())
	}
	span.SetAttributes(attribute.Int("afi.worker.batch_size", n))
	return n, nil
}
