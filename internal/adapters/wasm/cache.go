package wasm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
)

// ModuleCache compiles and reuses modules keyed by digest (or URI when digest empty).
type ModuleCache struct {
	mu   sync.Mutex
	mods map[string]*Module
	cfg  Config // defaults for compiled modules
}

// NewModuleCache builds an empty cache.
func NewModuleCache(cfg Config) *ModuleCache {
	return &ModuleCache{mods: map[string]*Module{}, cfg: cfg.withDefaults()}
}

// Close closes all cached modules.
func (c *ModuleCache) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	for k, m := range c.mods {
		if e := m.Close(ctx); e != nil && err == nil {
			err = e
		}
		delete(c.mods, k)
	}
	return err
}

// Get loads module bytes from uri, verifies digest when set, compiles once per cache key.
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
	c.mu.Unlock()

	path, err := ResolveModulePath(moduleURI)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("wasm: read %s: %w", path, err)
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

func cacheKey(uri, digest string) string {
	digest = strings.TrimSpace(strings.ToLower(digest))
	if digest != "" {
		return "sha256:" + digest
	}
	return "uri:" + strings.TrimSpace(uri)
}

// ResolveModulePath maps file:// URIs and plain paths to a filesystem path.
func ResolveModulePath(moduleURI string) (string, error) {
	uri := strings.TrimSpace(moduleURI)
	if uri == "" {
		return "", fmt.Errorf("wasm: empty module_uri")
	}
	if strings.Contains(uri, "://") {
		u, err := url.Parse(uri)
		if err != nil {
			return "", fmt.Errorf("wasm: module_uri: %w", err)
		}
		switch u.Scheme {
		case "file":
			if u.Path == "" {
				return "", fmt.Errorf("wasm: empty file path")
			}
			return u.Path, nil
		default:
			return "", fmt.Errorf("wasm: unsupported module_uri scheme %q (use file:// or a local path)", u.Scheme)
		}
	}
	return uri, nil
}

// VerifyDigest checks sha256 hex of data when want is non-empty.
func VerifyDigest(data []byte, want string) error {
	want = strings.TrimSpace(strings.ToLower(want))
	if want == "" {
		return nil
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("wasm: digest mismatch: got %s want %s", got, want)
	}
	return nil
}

// DigestHex returns sha256 hex of data.
func DigestHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
