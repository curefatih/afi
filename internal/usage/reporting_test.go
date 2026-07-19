package usage

import "testing"

func TestClassifyProviderHealth(t *testing.T) {
	t.Parallel()
	if ClassifyProviderHealth(0, 0, 0) != "unknown" {
		t.Fatal("unknown")
	}
	if ClassifyProviderHealth(10, 0, 0) != "healthy" {
		t.Fatal("healthy")
	}
	if ClassifyProviderHealth(10, 3, 0.3) != "degraded" {
		t.Fatal("degraded")
	}
	if ClassifyProviderHealth(10, 10, 1) != "down" {
		t.Fatal("down")
	}
}
