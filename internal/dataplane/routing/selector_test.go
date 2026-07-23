package routing

import (
	"math/rand"
	"testing"
)

func TestOrderedSelectorIdentity(t *testing.T) {
	t.Parallel()
	in := []Candidate{
		{ProviderID: "a", TargetModel: "m", Weight: 1},
		{ProviderID: "b", TargetModel: "m", Weight: 9},
	}
	out := OrderedSelector{}.Order(in, nil)
	if len(out) != 2 || out[0].ProviderID != "a" || out[1].ProviderID != "b" {
		t.Fatalf("out=%+v", out)
	}
}

func TestWeightedSelectorRemainderKeepsConfigOrder(t *testing.T) {
	t.Parallel()
	cands := []Candidate{
		{ProviderID: "a", Weight: 1},
		{ProviderID: "b", Weight: 1},
		{ProviderID: "c", Weight: 1},
	}
	for seed := int64(0); seed < 500; seed++ {
		out := WeightedSelector{}.Order(cands, rand.New(rand.NewSource(seed)))
		if out[0].ProviderID != "b" {
			continue
		}
		if out[1].ProviderID != "a" || out[2].ProviderID != "c" {
			t.Fatalf("remainder=%+v seed=%d", out, seed)
		}
		return
	}
	t.Fatal("no seed picked b as first")
}

func TestWeightedSelectorDistribution(t *testing.T) {
	t.Parallel()
	cands := []Candidate{
		{ProviderID: "light", Weight: 1},
		{ProviderID: "heavy", Weight: 9},
	}
	rng := rand.New(rand.NewSource(42))
	counts := map[string]int{}
	const n = 2000
	for i := 0; i < n; i++ {
		out := WeightedSelector{}.Order(cands, rng)
		counts[out[0].ProviderID]++
	}
	if counts["heavy"] < 1500 || counts["light"] < 100 {
		t.Fatalf("counts=%v", counts)
	}
}

func TestForStrategy(t *testing.T) {
	t.Parallel()
	if _, ok := ForStrategy("weighted").(WeightedSelector); !ok {
		t.Fatal("expected WeightedSelector")
	}
	if _, ok := ForStrategy("").(OrderedSelector); !ok {
		t.Fatal("expected OrderedSelector")
	}
	if _, ok := ForStrategy("latency").(OrderedSelector); !ok {
		t.Fatal("unknown should be ordered")
	}
}
