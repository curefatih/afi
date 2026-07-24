package gatewayconfig

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

func TestNormalizeRoutingStrategy(t *testing.T) {
	t.Parallel()
	got, err := NormalizeRoutingStrategy("")
	if err != nil || got != RoutingOrdered {
		t.Fatalf("empty: got=%q err=%v", got, err)
	}
	got, err = NormalizeRoutingStrategy(" WEIGHTED ")
	if err != nil || got != RoutingWeighted {
		t.Fatalf("weighted: got=%q err=%v", got, err)
	}
	got, err = NormalizeRoutingStrategy("latency")
	if err != nil || got != RoutingLatency {
		t.Fatalf("latency: got=%q err=%v", got, err)
	}
	got, err = NormalizeRoutingStrategy("COST")
	if err != nil || got != RoutingCost {
		t.Fatalf("cost: got=%q err=%v", got, err)
	}
	_, err = NormalizeRoutingStrategy("affinity")
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("affinity err=%v", err)
	}
}

func TestNormalizeWeight(t *testing.T) {
	t.Parallel()
	got, err := NormalizeWeight(0)
	if err != nil || got != 1 {
		t.Fatalf("zero: got=%d err=%v", got, err)
	}
	got, err = NormalizeWeight(3)
	if err != nil || got != 3 {
		t.Fatalf("three: got=%d err=%v", got, err)
	}
	_, err = NormalizeWeight(-1)
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("neg err=%v", err)
	}
}

func TestNormalizeRouteRouting(t *testing.T) {
	t.Parallel()
	strategy, weight, fbs, err := NormalizeRouteRouting("weighted", 0, []RouteFallback{
		{ProviderID: " p2 ", TargetModel: " m2 ", Weight: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strategy != RoutingWeighted || weight != 1 {
		t.Fatalf("strategy=%q weight=%d", strategy, weight)
	}
	if len(fbs) != 1 || fbs[0].ProviderID != "p2" || fbs[0].TargetModel != "m2" || fbs[0].Weight != 1 {
		t.Fatalf("fallbacks=%+v", fbs)
	}
}

func TestNewRouteDefaultsOrdered(t *testing.T) {
	t.Parallel()
	r, err := NewRoute("r1", "o1", "m1", "p1", "m1", nil, nil, "", 0, timeNowUTC())
	if err != nil {
		t.Fatal(err)
	}
	if r.RoutingStrategy != RoutingOrdered || r.Weight != 1 {
		t.Fatalf("route=%+v", r)
	}
}
