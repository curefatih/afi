package workers

import "testing"

func TestComputeCostUSD(t *testing.T) {
	t.Parallel()
	// 1M input @ $0.15 + 500k output @ $0.60 = 0.15 + 0.30 = 0.45
	got := ComputeCostUSD(1_000_000, 500_000, 0.15, 0.60)
	if got == nil || *got < 0.449 || *got > 0.451 {
		t.Fatalf("got %v", got)
	}
	zero := ComputeCostUSD(0, 0, 0.15, 0.60)
	if zero == nil || *zero != 0 {
		t.Fatalf("zero tokens: %v", zero)
	}
}
