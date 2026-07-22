package secrets

import (
	"context"
	"fmt"

	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/snapshot"
)

// CredentialOpener resolves snapshot credentials to plaintext secrets.
type CredentialOpener interface {
	Open(ctx context.Context, c snapshot.Credential) (string, error)
}

// CredentialResolver opens env, encrypted_db, or vault credentials.
type CredentialResolver struct {
	Env   Env
	Box   *credentials.Box
	Vault Resolver // Multi or scheme-specific; required for storage_kind=vault
}

func (r CredentialResolver) Open(ctx context.Context, c snapshot.Credential) (string, error) {
	switch c.StorageKind {
	case snapshot.CredentialStorageEnv:
		return r.Env.Get(ctx, NormalizeEnvRef(c.SecretRef))
	case snapshot.CredentialStorageEncryptedDB:
		if r.Box == nil {
			return "", fmt.Errorf("credentials master key not configured")
		}
		return r.Box.Open(c.EncryptedPayload)
	case snapshot.CredentialStorageVault:
		if r.Vault == nil {
			return "", fmt.Errorf("vault secret resolver not configured")
		}
		return r.Vault.Get(ctx, c.SecretRef)
	default:
		return "", fmt.Errorf("unknown storage_kind %q", c.StorageKind)
	}
}

// NewCredentialResolver builds a resolver; box may be nil when only env credentials are used.
func NewCredentialResolver(box *credentials.Box) CredentialResolver {
	return CredentialResolver{Env: Env{}, Box: box, Vault: Multi{Env: Env{}}}
}

// WithVault returns a copy with a vault Multi (or other) resolver.
func (r CredentialResolver) WithVault(v Resolver) CredentialResolver {
	r.Vault = v
	return r
}
