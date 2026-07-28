package access

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

const (
	SigningKeyAlgorithmEd25519 = "ed25519"
	SigningKeyStatusActive     = "active"
	SigningKeyStatusDisabled   = "disabled"
)

// SigningKey is the write-model public verification key for service auth.
type SigningKey struct {
	ID             string    `json:"id"`
	KeyID          string    `json:"key_id"`
	ProjectID      string    `json:"project_id,omitempty"`
	OrganizationID string    `json:"organization_id"`
	EnvironmentID  string    `json:"environment_id,omitempty"`
	Name           string    `json:"name"`
	Algorithm      string    `json:"algorithm"`
	PublicKeyPEM   string    `json:"public_key_pem"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func NormalizeSigningKeyAlgorithm(v string) string {
	if strings.TrimSpace(v) == "" {
		return SigningKeyAlgorithmEd25519
	}
	return strings.ToLower(strings.TrimSpace(v))
}

func NormalizeSigningKeyStatus(v string) string {
	if strings.TrimSpace(v) == "" {
		return SigningKeyStatusActive
	}
	return strings.ToLower(strings.TrimSpace(v))
}

func ValidateSigningKeyStatus(v string) error {
	switch NormalizeSigningKeyStatus(v) {
	case SigningKeyStatusActive, SigningKeyStatusDisabled:
		return nil
	default:
		return fmt.Errorf("%w: unsupported status %q", kernel.ErrInvalidRequest, v)
	}
}

func ParseEd25519PublicKeyPEM(publicKeyPEM string) (ed25519.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(publicKeyPEM)))
	if block == nil {
		return nil, fmt.Errorf("%w: invalid PEM public key", kernel.ErrInvalidRequest)
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse public key: %v", kernel.ErrInvalidRequest, err)
	}
	pub, ok := pubAny.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: public key must be ed25519", kernel.ErrInvalidRequest)
	}
	return pub, nil
}

func ValidateSigningKeyAlgorithmAndMaterial(algorithm, publicKeyPEM string) error {
	switch NormalizeSigningKeyAlgorithm(algorithm) {
	case SigningKeyAlgorithmEd25519:
		_, err := ParseEd25519PublicKeyPEM(publicKeyPEM)
		return err
	default:
		return fmt.Errorf("%w: unsupported algorithm %q", kernel.ErrInvalidRequest, algorithm)
	}
}

func NewSigningKey(
	id, keyID, orgID, projectID, environmentID, name, algorithm, publicKeyPEM string,
	now time.Time,
) (*SigningKey, error) {
	id = strings.TrimSpace(id)
	keyID = strings.TrimSpace(keyID)
	orgID = strings.TrimSpace(orgID)
	name = strings.TrimSpace(name)
	publicKeyPEM = strings.TrimSpace(publicKeyPEM)
	if id == "" || keyID == "" || orgID == "" || name == "" {
		return nil, fmt.Errorf("%w: id, key_id, organization_id, and name required", kernel.ErrInvalidRequest)
	}
	if projectID == "" && environmentID != "" {
		return nil, fmt.Errorf("%w: environment requires a project", kernel.ErrInvalidRequest)
	}
	algorithm = NormalizeSigningKeyAlgorithm(algorithm)
	if err := ValidateSigningKeyAlgorithmAndMaterial(algorithm, publicKeyPEM); err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &SigningKey{
		ID:             id,
		KeyID:          keyID,
		ProjectID:      strings.TrimSpace(projectID),
		OrganizationID: orgID,
		EnvironmentID:  strings.TrimSpace(environmentID),
		Name:           name,
		Algorithm:      algorithm,
		PublicKeyPEM:   publicKeyPEM,
		Status:         SigningKeyStatusActive,
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}, nil
}

func CreateSigningKey(
	ctx context.Context,
	repo SigningKeyRepository,
	projects ProjectOrgChecker,
	envs EnvironmentProjectChecker,
	id, keyID, orgID, projectID, environmentID, name, algorithm, publicKeyPEM string,
) (*SigningKey, error) {
	k, err := NewSigningKey(id, keyID, orgID, projectID, environmentID, name, algorithm, publicKeyPEM, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if projectID != "" {
		if err := projects.ProjectBelongsToOrg(ctx, projectID, orgID); err != nil {
			return nil, err
		}
	}
	if environmentID != "" {
		if err := envs.EnvironmentBelongsToProject(ctx, environmentID, projectID, orgID); err != nil {
			return nil, err
		}
	}
	if err := repo.Insert(ctx, *k); err != nil {
		return nil, err
	}
	return k, nil
}

func UpdateSigningKey(ctx context.Context, repo SigningKeyRepository, id, name, status string) (*SigningKey, error) {
	if strings.TrimSpace(name) == "" && strings.TrimSpace(status) == "" {
		return nil, fmt.Errorf("%w: name or status required", kernel.ErrInvalidRequest)
	}
	if strings.TrimSpace(status) != "" {
		if err := ValidateSigningKeyStatus(status); err != nil {
			return nil, err
		}
		status = NormalizeSigningKeyStatus(status)
	}
	return repo.UpdateMeta(ctx, id, strings.TrimSpace(name), status)
}

func RotateSigningKey(ctx context.Context, repo SigningKeyRepository, id, publicKeyPEM string) (*SigningKey, error) {
	k, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ValidateSigningKeyAlgorithmAndMaterial(k.Algorithm, publicKeyPEM); err != nil {
		return nil, err
	}
	return repo.UpdatePublicKey(ctx, id, strings.TrimSpace(publicKeyPEM))
}
