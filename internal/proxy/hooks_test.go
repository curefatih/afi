package proxy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/config"
)

func TestHookRunnerTimeout(t *testing.T) {
	dir := t.TempDir()
	slow := filepath.Join(dir, "slow.js")
	if err := os.WriteFile(slow, []byte(`
	function onRequest(ctx) {
		const start = Date.now();
		while (Date.now() - start < 200) {}
		return ctx;
	}
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{HookSpecs: []config.HookSpec{
			{Name: "onRequest", Path: slow},
		},
			TimeoutMS: 50,
		}}

	runner, err := NewHookRunner(cfg)
	if err != nil {
		t.Fatal(err)
	}

	reqCtx := NewRequestContext("gpt-4", nil, []byte(`{"model":"gpt-4"}`))
	start := time.Now()
	runner.Run(context.Background(), HookOnRequest, reqCtx)
	if elapsed := time.Since(start); elapsed > 150*time.Millisecond {
		t.Fatalf("expected middleware to be short-circuited quickly, took %s", elapsed)
	}
}

func TestHookRunnerMutatesContext(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "tag.js")
	if err := os.WriteFile(script, []byte(`
	function onRequest(ctx) {
		ctx.metadata.tag = "test";
		return ctx;
	}
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Hooks: config.HooksConfig{TimeoutMS: 50, HookSpecs: []config.HookSpec{
			{Name: "onRequest", Path: script},
		}},
	}

	runner, err := NewHookRunner(cfg)
	if err != nil {
		t.Fatal(err)
	}

	reqCtx := NewRequestContext("gpt-4", nil, nil)
	runner.Run(context.Background(), HookOnRequest, reqCtx)
	if reqCtx.Metadata["tag"] != "test" {
		t.Fatalf("expected metadata tag=test, got %#v", reqCtx.Metadata)
	}
}
