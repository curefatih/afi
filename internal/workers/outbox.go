package workers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/curefatih/afi/internal/usage"
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
func ProcessOnce(ctx context.Context, src OutboxSource, sink UsageSink, prices PriceLookup) (int, error) {
	rows, err := src.ClaimBatch(ctx, 50)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, row := range rows {
		var payload UsagePayload
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			return n, err
		}
		var cost *float64
		if prices != nil {
			model := payload.TargetModel
			if model == "" {
				model = payload.Model
			}
			if payload.ProviderType != "" && model != "" {
				in, out, ok, err := prices.LookupModelPrice(ctx, payload.ProviderType, model)
				if err != nil {
					return n, err
				}
				if ok {
					cost = ComputeCostUSD(payload.PromptTokens, payload.CompletionTokens, in, out)
				}
			}
		}
		if err := sink.InsertUsage(ctx, payload, cost); err != nil {
			return n, err
		}
		if err := src.MarkProcessed(ctx, row.ID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
