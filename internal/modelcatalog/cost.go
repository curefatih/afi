package modelcatalog

import "github.com/curefatih/afi/internal/usage"

// EstimateCostUSD computes USD cost from catalog pricing and a usage event.
// Returns nil when the model is unknown or required pricing/metrics are missing.
func EstimateCostUSD(e usage.Event) *float64 {
	model := e.TargetModel
	if model == "" {
		model = e.Model
	}
	entry, ok := Lookup(e.ProviderType, model)
	if !ok {
		return nil
	}
	switch entry.Mode {
	case ModeAudioSpeech:
		chars, ok := metricFloat(e.Metrics, "characters")
		if !ok || entry.InputCostPerCharacter == nil {
			return nil
		}
		cost := chars * *entry.InputCostPerCharacter
		return &cost
	case ModeAudioTranscription:
		secs, ok := metricFloat(e.Metrics, "audio_seconds")
		if !ok || entry.InputCostPerSecond == nil {
			return nil
		}
		cost := secs * *entry.InputCostPerSecond
		return &cost
	default:
		// chat / messages / unknown treated as token pricing
		if entry.InputCostPerMTok == nil || entry.OutputCostPerMTok == nil {
			return nil
		}
		if e.PromptTokens == 0 && e.CompletionTokens == 0 {
			return nil
		}
		cost := (float64(e.PromptTokens)/1_000_000.0)**entry.InputCostPerMTok +
			(float64(e.CompletionTokens)/1_000_000.0)**entry.OutputCostPerMTok
		return &cost
	}
}

func metricFloat(metrics map[string]any, key string) (float64, bool) {
	if metrics == nil {
		return 0, false
	}
	v, ok := metrics[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}
