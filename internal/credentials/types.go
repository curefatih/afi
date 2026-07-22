package credentials

import (
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

const (
	StorageEnv         = snapshot.CredentialStorageEnv
	StorageEncryptedDB = snapshot.CredentialStorageEncryptedDB
	StorageVault       = snapshot.CredentialStorageVault

	StatusActive   = "active"
	StatusDisabled = "disabled"

	ScopeOrganization = snapshot.ScopeOrganization
	ScopeProject      = snapshot.ScopeProject
	ScopeAPIKey       = snapshot.ScopeAPIKey
)

// Credential is an org-owned upstream provider secret (env ref or encrypted value).
type Credential struct {
	ID               string    `json:"id"`
	OrganizationID   string    `json:"organization_id"`
	Name             string    `json:"name"`
	ProviderType     string    `json:"provider_type"`
	StorageKind      string    `json:"storage_kind"`
	SecretRef        string    `json:"secret_ref,omitempty"`
	EncryptedPayload []byte    `json:"-"`
	KeyVersion       int       `json:"key_version,omitempty"`
	Status           string    `json:"status"`
	HasSecret        bool      `json:"has_secret"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Assignment binds a credential to an organization or project scope.
type Assignment struct {
	ID             string    `json:"id"`
	CredentialID   string    `json:"credential_id"`
	OrganizationID string    `json:"organization_id"`
	ProviderType   string    `json:"provider_type"`
	ScopeType      string    `json:"scope_type"`
	ScopeID        string    `json:"scope_id"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by,omitempty"`
}

// Public strips non-API fields (already omitted via json tags) and sets HasSecret.
func (c Credential) Public() Credential {
	out := c
	has := c.HasSecret
	if (c.StorageKind == StorageEnv || c.StorageKind == StorageVault) && c.SecretRef != "" {
		has = true
	}
	if c.StorageKind == StorageEncryptedDB && len(c.EncryptedPayload) > 0 {
		has = true
	}
	out.EncryptedPayload = nil
	out.HasSecret = has
	if c.StorageKind == StorageEncryptedDB {
		out.SecretRef = ""
	}
	return out
}

// NewCredential validates and builds a credential entity (secret material applied by caller).
func NewCredential(id, orgID, name, providerType, storageKind, status string, now time.Time) (*Credential, error) {
	name = strings.TrimSpace(name)
	providerType = strings.TrimSpace(providerType)
	storageKind = strings.TrimSpace(storageKind)
	status = strings.TrimSpace(status)
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", kernel.ErrInvalidRequest)
	}
	if providerType == "" {
		return nil, fmt.Errorf("%w: provider_type is required", kernel.ErrInvalidRequest)
	}
	switch storageKind {
	case StorageEnv, StorageEncryptedDB, StorageVault:
	default:
		return nil, fmt.Errorf("%w: storage_kind must be %q, %q, or %q", kernel.ErrInvalidRequest, StorageEnv, StorageEncryptedDB, StorageVault)
	}
	if status == "" {
		status = StatusActive
	}
	switch status {
	case StatusActive, StatusDisabled:
	default:
		return nil, fmt.Errorf("%w: invalid status %q", kernel.ErrInvalidRequest, status)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &Credential{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		ProviderType:   providerType,
		StorageKind:    storageKind,
		Status:         status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// ValidateAssignmentScope validates scope_type for credential assignments.
func ValidateAssignmentScope(scopeType string) error {
	switch scopeType {
	case ScopeOrganization, ScopeProject, ScopeAPIKey:
		return nil
	default:
		return fmt.Errorf("%w: scope_type must be %q, %q, or %q", kernel.ErrInvalidRequest, ScopeOrganization, ScopeProject, ScopeAPIKey)
	}
}
