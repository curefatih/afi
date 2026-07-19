package secrets

import (
	"context"
	"fmt"
	"os"
)

// Resolver resolves secret references (env var names today; vault later).
type Resolver interface {
	Get(ctx context.Context, ref string) (string, error)
}

// Env resolves secrets from process environment variables.
type Env struct{}

func (Env) Get(_ context.Context, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty secret ref")
	}
	v := os.Getenv(ref)
	if v == "" {
		return "", fmt.Errorf("missing env %s", ref)
	}
	return v, nil
}

// Default is the process-env resolver.
func Default() Resolver { return Env{} }
