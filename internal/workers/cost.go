package workers

// ComputeCostUSD returns USD cost from per-million-token prices, or nil if prices are missing.
func ComputeCostUSD(promptTokens, completionTokens int64, inputPerMTok, outputPerMTok float64) *float64 {
	if inputPerMTok < 0 || outputPerMTok < 0 {
		return nil
	}
	cost := (float64(promptTokens)/1_000_000.0)*inputPerMTok +
		(float64(completionTokens)/1_000_000.0)*outputPerMTok
	return &cost
}
