package routing

import (
	"math/rand"
	"testing"
)

func TestMemorySignalStoreEWMA(t *testing.T) {
	t.Parallel()
	s := NewMemorySignalStore()
	s.Observe("p1", "m1", 100, false)
	ms, ok := s.LatencyEWMA("p1", "m1")
	if !ok || ms != 100 {
		t.Fatalf("first observe: ms=%v ok=%v", ms, ok)
	}
	s.Observe("p1", "m1", 200, false)
	ms, ok = s.LatencyEWMA("p1", "m1")
	want := ewmaAlpha*200 + (1-ewmaAlpha)*100
	if !ok || ms != want {
		t.Fatalf("ewma: got=%v want=%v", ms, want)
	}
	rate, ok := s.ErrorRate("p1", "m1")
	if !ok || rate != 0 {
		t.Fatalf("error rate after successes: %v ok=%v", rate, ok)
	}
	s.Observe("p1", "m1", 50, true)
	rate, ok = s.ErrorRate("p1", "m1")
	if !ok || rate != 1.0/3.0 {
		t.Fatalf("error rate: got=%v", rate)
	}
	if _, ok := s.LatencyEWMA("missing", "m"); ok {
		t.Fatal("expected miss")
	}
}

func TestLatencySelectorOrdersByEWMA(t *testing.T) {
	t.Parallel()
	store := NewMemorySignalStore()
	store.Observe("slow", "m", 500, false)
	store.Observe("fast", "m", 50, false)
	cands := []Candidate{
		{ProviderID: "slow", TargetModel: "m"},
		{ProviderID: "fast", TargetModel: "m"},
		{ProviderID: "unknown", TargetModel: "m"},
	}
	out := LatencySelector{Store: store}.Order(cands, nil)
	if out[0].ProviderID != "fast" || out[1].ProviderID != "unknown" || out[2].ProviderID != "slow" {
		// unknown gets median of [50,500] = 275, so order: fast(50), unknown(275), slow(500)
		t.Fatalf("out=%+v", out)
	}
}

func TestLatencySelectorNoSignalsKeepsOrder(t *testing.T) {
	t.Parallel()
	cands := []Candidate{
		{ProviderID: "a", TargetModel: "m"},
		{ProviderID: "b", TargetModel: "m"},
	}
	out := LatencySelector{Store: NewMemorySignalStore()}.Order(cands, nil)
	if out[0].ProviderID != "a" || out[1].ProviderID != "b" {
		t.Fatalf("out=%+v", out)
	}
	out = LatencySelector{Store: nil}.Order(cands, nil)
	if out[0].ProviderID != "a" {
		t.Fatalf("nil store: %+v", out)
	}
}

func TestLatencySelectorStableTies(t *testing.T) {
	t.Parallel()
	store := NewMemorySignalStore()
	store.Observe("a", "m", 100, false)
	store.Observe("b", "m", 100, false)
	cands := []Candidate{
		{ProviderID: "a", TargetModel: "m"},
		{ProviderID: "b", TargetModel: "m"},
	}
	out := LatencySelector{Store: store}.Order(cands, nil)
	if out[0].ProviderID != "a" || out[1].ProviderID != "b" {
		t.Fatalf("stable tie broken: %+v", out)
	}
}

func TestCostSelectorUnknownLast(t *testing.T) {
	t.Parallel()
	// gpt-4o-mini and claude models are in catalog; made-up model is not.
	cands := []Candidate{
		{ProviderID: "unk", ProviderType: "openai", TargetModel: "afi-not-a-real-model-xyz"},
		{ProviderID: "oai", ProviderType: "openai", TargetModel: "gpt-4o-mini"},
		{ProviderID: "ant", ProviderType: "anthropic", TargetModel: "claude-3-5-haiku-20241022"},
	}
	out := CostSelector{}.Order(cands, nil)
	if out[len(out)-1].ProviderID != "unk" {
		t.Fatalf("unknown should be last: %+v", out)
	}
	// Both known models should precede unknown; cheaper first among known.
	if out[0].ProviderID == "unk" {
		t.Fatalf("known should be first: %+v", out)
	}
}

func TestCostSelectorStableTies(t *testing.T) {
	t.Parallel()
	cands := []Candidate{
		{ProviderID: "a", ProviderType: "openai", TargetModel: "afi-tie-a"},
		{ProviderID: "b", ProviderType: "openai", TargetModel: "afi-tie-b"},
	}
	out := CostSelector{}.Order(cands, nil)
	if out[0].ProviderID != "a" || out[1].ProviderID != "b" {
		t.Fatalf("unknown stable: %+v", out)
	}
}

func TestForStrategyAdaptive(t *testing.T) {
	t.Parallel()
	store := NewMemorySignalStore()
	if _, ok := ForStrategy("weighted", store).(WeightedSelector); !ok {
		t.Fatal("expected WeightedSelector")
	}
	if _, ok := ForStrategy("", store).(OrderedSelector); !ok {
		t.Fatal("expected OrderedSelector")
	}
	if _, ok := ForStrategy("latency", store).(LatencySelector); !ok {
		t.Fatal("expected LatencySelector")
	}
	if _, ok := ForStrategy("latency", nil).(OrderedSelector); !ok {
		t.Fatal("nil signals latency → ordered")
	}
	if _, ok := ForStrategy("cost", store).(CostSelector); !ok {
		t.Fatal("expected CostSelector")
	}
	if _, ok := ForStrategy("nope", store).(OrderedSelector); !ok {
		t.Fatal("unknown → ordered")
	}
	_ = rand.New(rand.NewSource(1))
}
