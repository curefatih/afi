package workers

import (
	"context"

	"github.com/curefatih/afi/internal/modelcatalog"
	"github.com/curefatih/afi/internal/usage"
)

// ComputeCostUSD returns USD cost from per-million-token prices, or nil if prices are missing.
func ComputeCostUSD(promptTokens, completionTokens int64, inputPerMTok, outputPerMTok float64) *float64 {
	if inputPerMTok < 0 || outputPerMTok < 0 {
		return nil
	}
	cost := (float64(promptTokens)/1_000_000.0)*inputPerMTok +
		(float64(completionTokens)/1_000_000.0)*outputPerMTok
	return &cost
}

// estimateUsageCost prefers DB model_prices overrides, then the curated catalog
// (chat tokens, TTS characters, STT audio_seconds).
func estimateUsageCost(ctx context.Context, payload usage.Event, prices PriceLookup) (*float64, error) {
	model := payload.TargetModel
	if model == "" {
		model = payload.Model
	}
	if prices != nil && payload.ProviderType != "" && model != "" {
		in, out, ok, err := prices.LookupModelPrice(ctx, payload.ProviderType, model)
		if err != nil {
			return nil, err
		}
		if ok {
			return ComputeCostUSD(payload.PromptTokens, payload.CompletionTokens, in, out), nil
		}
	}
	return modelcatalog.EstimateCostUSD(payload), nil
}
