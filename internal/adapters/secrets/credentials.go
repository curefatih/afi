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

// CredentialResolver opens env or encrypted_db credentials.
type CredentialResolver struct {
	Env Env
	Box *credentials.Box
}

func (r CredentialResolver) Open(ctx context.Context, c snapshot.Credential) (string, error) {
	switch c.StorageKind {
	case snapshot.CredentialStorageEnv:
		return r.Env.Get(ctx, c.SecretRef)
	case snapshot.CredentialStorageEncryptedDB:
		if r.Box == nil {
			return "", fmt.Errorf("credentials master key not configured")
		}
		return r.Box.Open(c.EncryptedPayload)
	default:
		return "", fmt.Errorf("unknown storage_kind %q", c.StorageKind)
	}
}

// NewCredentialResolver builds a resolver; box may be nil when only env credentials are used.
func NewCredentialResolver(box *credentials.Box) CredentialResolver {
	return CredentialResolver{Env: Env{}, Box: box}
}
