package memory_test

import (
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/memory"
	"github.com/curefatih/afi/internal/identity"
)

func TestSSOStateStoreTakeOnce(t *testing.T) {
	t.Parallel()
	store := memory.NewSSOStateStore(time.Minute)
	_ = store.Put("abc", identity.SSOState{Provider: "google", ReturnTo: "/app"})
	got, ok := store.Take("abc")
	if !ok || got.Provider != "google" {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
	if _, ok := store.Take("abc"); ok {
		t.Fatal("expected consumed")
	}
}
