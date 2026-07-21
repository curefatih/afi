package wasm

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPoolReusesInstances(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "pool", Timeout: time.Second, PoolSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)

	hook, err := NewBeforeCall(mod)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		d, err := hook.BeforeCall(ctx, benchCallAllow())
		if err != nil || !d.Allow {
			t.Fatalf("i=%d allow=%v err=%v", i, d.Allow, err)
		}
	}
	// At most PoolSize instances should sit in the pool.
	if got := len(mod.pool); got > 2 {
		t.Fatalf("pool len=%d want <=2", got)
	}
}

func TestPoolConcurrent(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "pool-c", Timeout: time.Second, PoolSize: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)

	hook, err := NewBeforeCall(mod)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := hook.BeforeCall(ctx, benchCallAllow())
			if err != nil {
				errCh <- err
				return
			}
			if !d.Allow {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNoPoolStillWorks(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "nopool", Timeout: time.Second, PoolSize: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)
	hook, err := NewBeforeCall(mod)
	if err != nil {
		t.Fatal(err)
	}
	d, err := hook.BeforeCall(ctx, benchCallAllow())
	if err != nil || !d.Allow {
		t.Fatalf("allow=%v err=%v", d.Allow, err)
	}
}
