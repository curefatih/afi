package secrets

import (
	"context"
	"fmt"
	"strings"
)

// Multi routes Get calls by secret-ref scheme.
type Multi struct {
	Env       Resolver
	AWSSM     Resolver
	Hashicorp Resolver
}

func (m Multi) Get(ctx context.Context, ref string) (string, error) {
	p, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	switch p.Scheme {
	case "env", "":
		if m.Env == nil {
			return "", fmt.Errorf("env secret resolver not configured")
		}
		return m.Env.Get(ctx, p.Path)
	case "aws-sm":
		if m.AWSSM == nil {
			return "", fmt.Errorf("aws-sm secret resolver not configured")
		}
		return m.AWSSM.Get(ctx, ref)
	case "hashicorp":
		if m.Hashicorp == nil {
			return "", fmt.Errorf("hashicorp secret resolver not configured")
		}
		return m.Hashicorp.Get(ctx, ref)
	default:
		return "", fmt.Errorf("unsupported secret scheme %q", p.Scheme)
	}
}

// NormalizeEnvRef turns bare names into env paths for CredentialResolver.
func NormalizeEnvRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(strings.ToLower(ref), "env://") {
		p, err := ParseRef(ref)
		if err == nil {
			return p.Path
		}
	}
	return ref
}
