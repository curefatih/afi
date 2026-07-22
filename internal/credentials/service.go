package credentials

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

// Repository persists credentials and assignments.
type Repository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Credential, error)
	Get(ctx context.Context, id string) (*Credential, error)
	Insert(ctx context.Context, c Credential) error
	UpdateMeta(ctx context.Context, id, name, status string) (*Credential, error)
	UpdateSecret(ctx context.Context, id string, secretRef string, payload []byte, keyVersion int) (*Credential, error)
	Delete(ctx context.Context, id string) error
	OrgID(ctx context.Context, id string) (string, error)
	HasAssignments(ctx context.Context, credentialID string) (bool, error)

	ListAssignmentsByOrg(ctx context.Context, orgID string) ([]Assignment, error)
	InsertAssignment(ctx context.Context, a Assignment) error
	DeleteAssignment(ctx context.Context, id string) error
	AssignmentOrgID(ctx context.Context, id string) (string, error)
	UpsertAssignment(ctx context.Context, a Assignment) (*Assignment, error)
}

// ScopeChecker verifies assignment targets belong to the credential's org.
type ScopeChecker interface {
	ProjectBelongsToOrg(ctx context.Context, projectID, orgID string) error
	APIKeyBelongsToOrg(ctx context.Context, keyID, orgID string) error
}

// CreateInput is the write payload for a new credential.
type CreateInput struct {
	ID           string
	OrgID        string
	Name         string
	ProviderType string
	StorageKind  string
	SecretRef    string
	SecretValue  string
	Now          time.Time
}

// Create validates, optionally encrypts, and persists a credential.
func Create(ctx context.Context, repo Repository, box *Box, in CreateInput) (*Credential, error) {
	c, err := NewCredential(in.ID, in.OrgID, in.Name, in.ProviderType, in.StorageKind, StatusActive, in.Now)
	if err != nil {
		return nil, err
	}
	switch c.StorageKind {
	case StorageEnv, StorageVault:
		ref := strings.TrimSpace(in.SecretRef)
		if ref == "" {
			kind := "env"
			if c.StorageKind == StorageVault {
				kind = "vault"
			}
			return nil, fmt.Errorf("%w: secret_ref is required for %s storage", kernel.ErrInvalidRequest, kind)
		}
		c.SecretRef = ref
		c.HasSecret = true
	case StorageEncryptedDB:
		val := strings.TrimSpace(in.SecretValue)
		if val == "" {
			return nil, fmt.Errorf("%w: secret_value is required for encrypted_db storage", kernel.ErrInvalidRequest)
		}
		if box == nil {
			return nil, fmt.Errorf("%w: credentials master key not configured", kernel.ErrInvalidRequest)
		}
		payload, err := box.Seal(val)
		if err != nil {
			return nil, err
		}
		c.EncryptedPayload = payload
		c.KeyVersion = CurrentKeyVersion
		c.HasSecret = true
	}
	if err := repo.Insert(ctx, *c); err != nil {
		return nil, err
	}
	pub := c.Public()
	return &pub, nil
}

// RotateSecret replaces the secret material for an existing credential.
func RotateSecret(ctx context.Context, repo Repository, box *Box, id, secretRef, secretValue string) (*Credential, error) {
	c, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	switch c.StorageKind {
	case StorageEnv, StorageVault:
		ref := strings.TrimSpace(secretRef)
		if ref == "" {
			return nil, fmt.Errorf("%w: secret_ref is required", kernel.ErrInvalidRequest)
		}
		updated, err := repo.UpdateSecret(ctx, id, ref, nil, 0)
		if err != nil {
			return nil, err
		}
		pub := updated.Public()
		return &pub, nil
	case StorageEncryptedDB:
		val := strings.TrimSpace(secretValue)
		if val == "" {
			return nil, fmt.Errorf("%w: secret_value is required", kernel.ErrInvalidRequest)
		}
		if box == nil {
			return nil, fmt.Errorf("%w: credentials master key not configured", kernel.ErrInvalidRequest)
		}
		payload, err := box.Seal(val)
		if err != nil {
			return nil, err
		}
		updated, err := repo.UpdateSecret(ctx, id, "", payload, CurrentKeyVersion)
		if err != nil {
			return nil, err
		}
		pub := updated.Public()
		return &pub, nil
	default:
		return nil, fmt.Errorf("%w: unknown storage_kind", kernel.ErrInvalidRequest)
	}
}

// AssignInput creates or replaces the credential slot for a scope.
type AssignInput struct {
	ID           string
	CredentialID string
	ScopeType    string
	ScopeID      string
	CreatedBy    string
	Now          time.Time
}

// Assign binds a credential to org or project scope (one slot per provider_type).
func Assign(ctx context.Context, repo Repository, scopes ScopeChecker, in AssignInput) (*Assignment, error) {
	if err := ValidateAssignmentScope(in.ScopeType); err != nil {
		return nil, err
	}
	scopeID := strings.TrimSpace(in.ScopeID)
	if scopeID == "" {
		return nil, fmt.Errorf("%w: scope_id is required", kernel.ErrInvalidRequest)
	}
	c, err := repo.Get(ctx, in.CredentialID)
	if err != nil {
		return nil, err
	}
	if c.Status != StatusActive {
		return nil, fmt.Errorf("%w: credential is not active", kernel.ErrInvalidRequest)
	}
	switch in.ScopeType {
	case ScopeOrganization:
		if scopeID != c.OrganizationID {
			return nil, fmt.Errorf("%w: organization scope_id must match credential organization", kernel.ErrInvalidRequest)
		}
	case ScopeProject:
		if err := scopes.ProjectBelongsToOrg(ctx, scopeID, c.OrganizationID); err != nil {
			return nil, err
		}
	case ScopeAPIKey:
		if err := scopes.APIKeyBelongsToOrg(ctx, scopeID, c.OrganizationID); err != nil {
			return nil, err
		}
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	a := Assignment{
		ID:             in.ID,
		CredentialID:   c.ID,
		OrganizationID: c.OrganizationID,
		ProviderType:   c.ProviderType,
		ScopeType:      in.ScopeType,
		ScopeID:        scopeID,
		CreatedAt:      now.UTC(),
		CreatedBy:      in.CreatedBy,
	}
	return repo.UpsertAssignment(ctx, a)
}

// DeleteCredential removes a credential when it has no assignments.
func DeleteCredential(ctx context.Context, repo Repository, id string) error {
	has, err := repo.HasAssignments(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return fmt.Errorf("%w: credential still has assignments", kernel.ErrConflict)
	}
	return repo.Delete(ctx, id)
}
