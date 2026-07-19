package workers

import (
	"context"
	"encoding/json"
	"time"
)

type OutboxRow struct {
	ID          int64
	Payload     []byte
	ProcessedAt *time.Time
}

type UsagePayload struct {
	OrganizationID   string         `json:"organization_id"`
	ProjectID        string         `json:"project_id"`
	APIKeyID         string         `json:"api_key_id"`
	Model            string         `json:"model"`
	ProviderType     string         `json:"provider_type"`
	TargetModel      string         `json:"target_model"`
	Status           string         `json:"status"`
	LatencyMs        int64          `json:"latency_ms"`
	PromptTokens     int64          `json:"prompt_tokens"`
	CompletionTokens int64          `json:"completion_tokens"`
	Modality         string         `json:"modality"`
	Metrics          map[string]any `json:"metrics,omitempty"`
}

type OutboxSource interface {
	ClaimBatch(ctx context.Context, limit int) ([]OutboxRow, error)
	MarkProcessed(ctx context.Context, id int64) error
}

type PriceLookup interface {
	LookupModelPrice(ctx context.Context, providerType, model string) (inputPerMTok, outputPerMTok float64, ok bool, err error)
}

type UsageSink interface {
	InsertUsage(ctx context.Context, e UsagePayload, costUSD *float64) error
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
