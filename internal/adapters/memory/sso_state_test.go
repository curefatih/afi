package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/memory"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

func TestSSOStateStoreTakeOnce(t *testing.T) {
	t.Parallel()
	store := memory.NewSSOStateStore(time.Minute)
	ctx := context.Background()
	_ = store.Put(ctx, "abc", identity.SSOState{Provider: "google", ReturnTo: "/app"})
	got, err := store.Take(ctx, "abc")
	if err != nil || got.Provider != "google" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if _, err := store.Take(ctx, "abc"); !errors.Is(err, kernel.ErrNotFound) {
		t.Fatalf("expected consumed, got %v", err)
	}
}
