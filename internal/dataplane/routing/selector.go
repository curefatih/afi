package routing

import (
	"math/rand"
	"strings"
)

// Strategy values mirrored from gatewayconfig / snapshot.
const (
	StrategyOrdered  = "ordered"
	StrategyWeighted = "weighted"
)

// Candidate is a provider+model target considered for an attempt list.
type Candidate struct {
	ProviderID  string
	TargetModel string
	Weight      int
}

// Selector orders candidates for the first attempt and subsequent failovers.
type Selector interface {
	Order(candidates []Candidate, rng *rand.Rand) []Candidate
}

// OrderedSelector keeps config order (primary then fallbacks).
type OrderedSelector struct{}

func (OrderedSelector) Order(candidates []Candidate, _ *rand.Rand) []Candidate {
	return cloneCandidates(candidates)
}

// WeightedSelector picks the first attempt by weight; remainder keep config order.
type WeightedSelector struct{}

func (WeightedSelector) Order(candidates []Candidate, rng *rand.Rand) []Candidate {
	if len(candidates) <= 1 {
		return cloneCandidates(candidates)
	}
	total := 0
	for _, c := range candidates {
		w := c.Weight
		if w < 1 {
			w = 1
		}
		total += w
	}
	pick := 0
	if total > 0 {
		n := intn(rng, total)
		acc := 0
		for i, c := range candidates {
			w := c.Weight
			if w < 1 {
				w = 1
			}
			acc += w
			if n < acc {
				pick = i
				break
			}
		}
	}
	out := make([]Candidate, 0, len(candidates))
	out = append(out, candidates[pick])
	for i, c := range candidates {
		if i == pick {
			continue
		}
		out = append(out, c)
	}
	return out
}

// SignalStore is reserved for adaptive latency/cost routing. Not used by ordered/weighted.
type SignalStore interface {
	Observe(providerID, targetModel string, latencyMs int64, failed bool)
	LatencyEWMA(providerID, targetModel string) (ms float64, ok bool)
	ErrorRate(providerID, targetModel string) (rate float64, ok bool)
}

// ForStrategy returns the selector for a routing strategy (unknown → ordered).
func ForStrategy(strategy string) Selector {
	switch strings.TrimSpace(strings.ToLower(strategy)) {
	case StrategyWeighted:
		return WeightedSelector{}
	default:
		return OrderedSelector{}
	}
}

func cloneCandidates(in []Candidate) []Candidate {
	if len(in) == 0 {
		return nil
	}
	out := make([]Candidate, len(in))
	copy(out, in)
	return out
}

func intn(rng *rand.Rand, n int) int {
	if n <= 0 {
		return 0
	}
	if rng != nil {
		return rng.Intn(n)
	}
	return rand.Intn(n)
}
