package access

import (
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// APIKey is the write-model virtual API key.
type APIKey struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id,omitempty"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"`
	OwnerUserID    string    `json:"owner_user_id,omitempty"`
	KeyPrefix      string    `json:"key_prefix"`
	Key            string    `json:"key,omitempty"` // plaintext only on create
	CreatedAt      time.Time `json:"created_at"`
}

// Prefix returns a short display prefix for an API key.
func Prefix(raw string) string {
	const n = 10
	if len(raw) <= n {
		return raw
	}
	return raw[:n]
}

// Hash returns the persisted key hash.
func Hash(raw string) string {
	return snapshot.HashKey(raw)
}

// NormalizeKind defaults empty kind to service_account.
func NormalizeKind(kind string) string {
	if kind == "" {
		return snapshot.KeyKindServiceAccount
	}
	return kind
}

// ValidateKindRules enforces personal vs service_account invariants.
func ValidateKindRules(kind, ownerUserID, projectID string) error {
	k, err := ParseKeyKind(kind)
	if err != nil {
		return err
	}
	switch k {
	case KeyKindPersonal:
		if ownerUserID == "" {
			return fmt.Errorf("%w: personal keys require owner", kernel.ErrInvalidRequest)
		}
		if projectID != "" {
			return fmt.Errorf("%w: personal keys cannot have a project", kernel.ErrInvalidRequest)
		}
	case KeyKindServiceAccount:
		if ownerUserID != "" {
			return fmt.Errorf("%w: service account keys cannot have an owner", kernel.ErrInvalidRequest)
		}
	}
	return nil
}

// NewAPIKey builds a validated key entity (does not check project membership).
func NewAPIKey(id, orgID, kind, ownerUserID, projectID, name, rawKey string, now time.Time) (*APIKey, error) {
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if rawKey == "" {
		return nil, fmt.Errorf("%w: raw key required", kernel.ErrInvalidRequest)
	}
	kind = NormalizeKind(kind)
	if err := ValidateKindRules(kind, ownerUserID, projectID); err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &APIKey{
		ID:             id,
		ProjectID:      projectID,
		OrganizationID: orgID,
		Name:           name,
		Kind:           kind,
		OwnerUserID:    ownerUserID,
		KeyPrefix:      Prefix(rawKey),
		Key:            rawKey,
		CreatedAt:      now.UTC(),
	}, nil
}
