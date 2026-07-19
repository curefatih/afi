package controlplane

import "testing"

func TestClassifyProviderHealth(t *testing.T) {
	t.Parallel()
	if classifyProviderHealth(0, 0, 0) != "unknown" {
		t.Fatal("unknown")
	}
	if classifyProviderHealth(10, 0, 0) != "healthy" {
		t.Fatal("healthy")
	}
	if classifyProviderHealth(10, 3, 0.3) != "degraded" {
		t.Fatal("degraded")
	}
	if classifyProviderHealth(10, 10, 1) != "down" {
		t.Fatal("down")
	}
}
