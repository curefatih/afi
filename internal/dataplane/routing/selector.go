package routing

import (
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/curefatih/afi/internal/modelcatalog"
)

// Strategy values mirrored from gatewayconfig / snapshot.
const (
	StrategyOrdered  = "ordered"
	StrategyWeighted = "weighted"
	StrategyLatency  = "latency"
	StrategyCost     = "cost"
)

// Candidate is a provider+model target considered for an attempt list.
type Candidate struct {
	ProviderID   string
	ProviderType string
	TargetModel  string
	Weight       int
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

// LatencySelector orders by ascending EWMA latency; unknowns get the median of known.
type LatencySelector struct {
	Store SignalStore
}

func (s LatencySelector) Order(candidates []Candidate, _ *rand.Rand) []Candidate {
	if len(candidates) <= 1 || s.Store == nil {
		return cloneCandidates(candidates)
	}
	type scored struct {
		c     Candidate
		idx   int
		score float64
		known bool
	}
	items := make([]scored, len(candidates))
	known := make([]float64, 0, len(candidates))
	for i, c := range candidates {
		ms, ok := s.Store.LatencyEWMA(c.ProviderID, c.TargetModel)
		items[i] = scored{c: c, idx: i, score: ms, known: ok}
		if ok {
			known = append(known, ms)
		}
	}
	if len(known) == 0 {
		return cloneCandidates(candidates)
	}
	sort.Float64s(known)
	median := known[len(known)/2]
	if len(known)%2 == 0 {
		median = (known[len(known)/2-1] + known[len(known)/2]) / 2
	}
	for i := range items {
		if !items[i].known {
			items[i].score = median
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].idx < items[j].idx
		}
		return items[i].score < items[j].score
	})
	out := make([]Candidate, len(items))
	for i, it := range items {
		out[i] = it.c
	}
	return out
}

// CostSelector orders by ascending catalog unit price; unknown prices last.
type CostSelector struct{}

func (CostSelector) Order(candidates []Candidate, _ *rand.Rand) []Candidate {
	if len(candidates) <= 1 {
		return cloneCandidates(candidates)
	}
	type scored struct {
		c     Candidate
		idx   int
		score float64
		known bool
	}
	items := make([]scored, len(candidates))
	for i, c := range candidates {
		score, ok := catalogUnitScore(c.ProviderType, c.TargetModel)
		items[i] = scored{c: c, idx: i, score: score, known: ok}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].known != items[j].known {
			return items[i].known
		}
		if !items[i].known {
			return items[i].idx < items[j].idx
		}
		if items[i].score == items[j].score {
			return items[i].idx < items[j].idx
		}
		return items[i].score < items[j].score
	})
	out := make([]Candidate, len(items))
	for i, it := range items {
		out[i] = it.c
	}
	return out
}

// catalogUnitScore returns InputCostPerMTok + OutputCostPerMTok (nil as 0).
// Both nil → unknown.
func catalogUnitScore(providerType, targetModel string) (float64, bool) {
	e, ok := modelcatalog.Lookup(providerType, targetModel)
	if !ok {
		return 0, false
	}
	var sum float64
	has := false
	if e.InputCostPerMTok != nil {
		sum += *e.InputCostPerMTok
		has = true
	}
	if e.OutputCostPerMTok != nil {
		sum += *e.OutputCostPerMTok
		has = true
	}
	if !has {
		return 0, false
	}
	if math.IsNaN(sum) || math.IsInf(sum, 0) {
		return 0, false
	}
	return sum, true
}

// SignalStore records and queries gateway-local upstream signals for adaptive routing.
type SignalStore interface {
	Observe(providerID, targetModel string, latencyMs int64, failed bool)
	LatencyEWMA(providerID, targetModel string) (ms float64, ok bool)
	ErrorRate(providerID, targetModel string) (rate float64, ok bool)
}

// ForStrategy returns the selector for a routing strategy (unknown → ordered).
// signals is used by latency; nil signals fall back to ordered for latency.
func ForStrategy(strategy string, signals SignalStore) Selector {
	switch strings.TrimSpace(strings.ToLower(strategy)) {
	case StrategyWeighted:
		return WeightedSelector{}
	case StrategyLatency:
		if signals == nil {
			return OrderedSelector{}
		}
		return LatencySelector{Store: signals}
	case StrategyCost:
		return CostSelector{}
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
