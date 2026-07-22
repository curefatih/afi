package wasm

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Get loads module bytes from uri, verifies digest when set, compiles once per cache key.
// Supported URIs: local path, file://, s3://bucket/key (when Fetcher is set).
func (c *ModuleCache) Get(ctx context.Context, moduleURI, digest, name string) (*Module, error) {
	if c == nil {
		return nil, fmt.Errorf("wasm: nil cache")
	}
	key := cacheKey(moduleURI, digest)
	c.mu.Lock()
	if m, ok := c.mods[key]; ok {
		c.mu.Unlock()
		return m, nil
	}
	fetcher := c.fetcher
	c.mu.Unlock()

	b, err := c.loadBytes(ctx, moduleURI, fetcher)
	if err != nil {
		return nil, err
	}
	if err := VerifyDigest(b, digest); err != nil {
		return nil, err
	}
	cfg := c.cfg
	if name != "" {
		cfg.Name = name
	} else if cfg.Name == "" {
		cfg.Name = "wasm:" + key
	}
	mod, err := Compile(ctx, b, cfg)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.mods[key]; ok {
		_ = mod.Close(ctx)
		return existing, nil
	}
	c.mods[key] = mod
	return mod, nil
}

// SetFetcher attaches a remote blob fetcher (e.g. S3). Nil clears it.
func (c *ModuleCache) SetFetcher(f BlobFetcher) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.fetcher = f
	c.mu.Unlock()
}

func (c *ModuleCache) loadBytes(ctx context.Context, moduleURI string, fetcher BlobFetcher) ([]byte, error) {
	uri := strings.TrimSpace(moduleURI)
	if strings.HasPrefix(uri, "s3://") {
		if fetcher == nil {
			return nil, fmt.Errorf("wasm: s3:// uri requires gateway S3 config")
		}
		return fetcher.Fetch(ctx, uri)
	}
	path, err := ResolveModulePath(uri)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("wasm: read %s: %w", path, err)
	}
	return b, nil
}
