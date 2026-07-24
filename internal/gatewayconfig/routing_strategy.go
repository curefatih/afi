package gatewayconfig

import (
	"fmt"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
)

// Routing strategy values persisted on routes and compiled into snapshots.
const (
	RoutingOrdered  = "ordered"
	RoutingWeighted = "weighted"
	RoutingLatency  = "latency"
	RoutingCost     = "cost"
)

// NormalizeRoutingStrategy trims/lowercases strategy; empty becomes ordered.
func NormalizeRoutingStrategy(strategy string) (string, error) {
	strategy = strings.TrimSpace(strings.ToLower(strategy))
	if strategy == "" {
		return RoutingOrdered, nil
	}
	switch strategy {
	case RoutingOrdered, RoutingWeighted, RoutingLatency, RoutingCost:
		return strategy, nil
	default:
		return "", fmt.Errorf("%w: routing_strategy must be ordered, weighted, latency, or cost", kernel.ErrInvalidRequest)
	}
}

// NormalizeWeight returns a positive weight; omitted/zero defaults to 1.
func NormalizeWeight(w int) (int, error) {
	if w < 0 {
		return 0, fmt.Errorf("%w: weight must be >= 0", kernel.ErrInvalidRequest)
	}
	if w == 0 {
		return 1, nil
	}
	return w, nil
}

// NormalizeRouteRouting normalizes strategy and primary/fallback weights.
func NormalizeRouteRouting(strategy string, weight int, fallbacks []RouteFallback) (string, int, []RouteFallback, error) {
	strategy, err := NormalizeRoutingStrategy(strategy)
	if err != nil {
		return "", 0, nil, err
	}
	weight, err = NormalizeWeight(weight)
	if err != nil {
		return "", 0, nil, err
	}
	if fallbacks == nil {
		fallbacks = []RouteFallback{}
	}
	out := make([]RouteFallback, len(fallbacks))
	for i, fb := range fallbacks {
		w, err := NormalizeWeight(fb.Weight)
		if err != nil {
			return "", 0, nil, fmt.Errorf("fallbacks[%d]: %w", i, err)
		}
		out[i] = RouteFallback{
			ProviderID:  strings.TrimSpace(fb.ProviderID),
			TargetModel: strings.TrimSpace(fb.TargetModel),
			Weight:      w,
		}
	}
	return strategy, weight, out, nil
}
