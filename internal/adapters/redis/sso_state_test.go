package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	afiredis "github.com/curefatih/afi/internal/adapters/redis"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/redis/go-redis/v9"
)

func TestSSOStateStoreRoundTrip(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	store := afiredis.NewSSOStateStore(rdb, time.Minute)
	ctx := context.Background()
	err := store.Put(ctx, "st1", identity.SSOState{
		Provider: "okta", ReturnTo: "/app", ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Take(ctx, "st1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "okta" || got.ReturnTo != "/app" {
		t.Fatalf("%+v", got)
	}
	if _, err := store.Take(ctx, "st1"); !errors.Is(err, kernel.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}
