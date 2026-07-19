package controlplane

import "testing"

func TestKeyPrefix(t *testing.T) {
	t.Parallel()
	if got := KeyPrefix("sk-abcdefghijklmnop"); got != "sk-abcdefg" {
		t.Fatalf("got %q", got)
	}
	if got := KeyPrefix("short"); got != "short" {
		t.Fatalf("got %q", got)
	}
}
