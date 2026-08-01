package controlplane

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/curefatih/afi/internal/telemetry"
	"github.com/curefatih/afi/internal/usage"
)

// UsageReportObserver accepts spoke/regional usage reports for observation only.
// It does not write usage_events or usage_outbox — durable analytics stay regional
// or are exported via OTel from these counters/logs.
type UsageReportObserver struct {
	Log     *slog.Logger
	Metrics *telemetry.ControlPlaneMetrics
}

func NewUsageReportObserver(log *slog.Logger) *UsageReportObserver {
	return &UsageReportObserver{Log: log}
}

// Enqueue implements the spoke usage ingest port without persistence.
func (o *UsageReportObserver) Enqueue(ctx context.Context, payload []byte) error {
	var e usage.Event
	if err := json.Unmarshal(payload, &e); err != nil {
		return err
	}
	region := ""
	if e.Tags != nil {
		region = e.Tags["region"]
	}
	modality := usage.NormalizeModality(e.Modality)
	if o != nil && o.Metrics != nil {
		o.Metrics.RecordSpokeUsage(ctx, region, e.Status, modality, e.PromptTokens, e.CompletionTokens)
	}
	if o != nil && o.Log != nil {
		depID := ""
		if e.Tags != nil {
			depID = e.Tags["deployment_id"]
		}
		o.Log.Info("spoke usage report",
			"region", region,
			"org", e.OrganizationID,
			"model", e.Model,
			"provider", e.ProviderType,
			"status", e.Status,
			"modality", modality,
			"prompt_tokens", e.PromptTokens,
			"completion_tokens", e.CompletionTokens,
			"latency_ms", e.LatencyMs,
			"deployment_id", depID,
		)
	}
	return nil
}
