package wasm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAndDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "h.wasm")
	data := []byte{0x00, 0x61, 0x73, 0x6d}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveModulePath("file://" + path)
	if err != nil || got != path {
		t.Fatalf("path=%q err=%v", got, err)
	}
	sum := DigestHex(data)
	if err := VerifyDigest(data, sum); err != nil {
		t.Fatal(err)
	}
	if err := VerifyDigest(data, "deadbeef"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestModuleCacheGet(t *testing.T) {
	ctx := context.Background()
	path := exampleWASMPath(t)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dig := DigestHex(b)
	cache := NewModuleCache(Config{Name: "cache-test", Timeout: 0})
	defer cache.Close(ctx)

	m1, err := cache.Get(ctx, path, dig, "a")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := cache.Get(ctx, path, dig, "b")
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 {
		t.Fatal("expected cached module reuse")
	}
}
