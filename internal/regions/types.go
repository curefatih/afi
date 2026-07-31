package regions

import (
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

const (
	RegionStatusActive    = "active"
	RegionStatusDraining  = "draining"
	RegionStatusDisabled  = "disabled"

	DeploymentStatusPending  = "pending"
	DeploymentStatusHealthy  = "healthy"
	DeploymentStatusStale    = "stale"
	DeploymentStatusDisabled = "disabled"

	// DefaultHeartbeatTTL marks a healthy deployment stale when last_seen exceeds this.
	DefaultHeartbeatTTL = 2 * time.Minute

	maxRegionSlugLen = 64
)

// Region is a logical isolation boundary for gateway deployments.
type Region struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GatewayDeployment is a registered spoke gateway (or HA group) in a region.
type GatewayDeployment struct {
	ID                       string     `json:"id"`
	RegionID                 string     `json:"region_id"`
	Name                     string     `json:"name"`
	PublicBaseURL            string     `json:"public_base_url,omitempty"`
	JoinTokenHash            string     `json:"-"`
	Status                   string     `json:"status"`
	LastSeenAt               *time.Time `json:"last_seen_at,omitempty"`
	ReportedSnapshotVersion  int64      `json:"reported_snapshot_version"`
	ReportedBuild            string     `json:"reported_build,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// DeploymentWithToken is returned once when a join token is minted.
type DeploymentWithToken struct {
	Deployment GatewayDeployment `json:"deployment"`
	JoinToken  string            `json:"join_token"`
}

// NormalizeRegionSlug lowercases and trims a slug.
func NormalizeRegionSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

// ValidateRegionSlug ensures slug is lowercase alnum with -/_ and length ≤ 64.
func ValidateRegionSlug(slug string) error {
	slug = NormalizeRegionSlug(slug)
	if slug == "" {
		return fmt.Errorf("%w: slug is required", kernel.ErrInvalidRequest)
	}
	if len(slug) > maxRegionSlugLen {
		return fmt.Errorf("%w: slug must be at most %d characters", kernel.ErrInvalidRequest, maxRegionSlugLen)
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("%w: slug must be lowercase alphanumeric with - or _", kernel.ErrInvalidRequest)
	}
	return nil
}

// ValidateRegionStatus checks status is a known region status.
func ValidateRegionStatus(status string) error {
	switch status {
	case RegionStatusActive, RegionStatusDraining, RegionStatusDisabled:
		return nil
	default:
		return fmt.Errorf("%w: status must be active, draining, or disabled", kernel.ErrInvalidRequest)
	}
}

// ValidateDeploymentStatus checks status is a known deployment status.
func ValidateDeploymentStatus(status string) error {
	switch status {
	case DeploymentStatusPending, DeploymentStatusHealthy, DeploymentStatusStale, DeploymentStatusDisabled:
		return nil
	default:
		return fmt.Errorf("%w: status must be pending, healthy, stale, or disabled", kernel.ErrInvalidRequest)
	}
}

// NewRegion builds a validated region entity.
func NewRegion(id, slug, name string, now time.Time) (*Region, error) {
	name = strings.TrimSpace(name)
	slug = NormalizeRegionSlug(slug)
	if id == "" {
		return nil, fmt.Errorf("%w: id required", kernel.ErrInvalidRequest)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	if err := ValidateRegionSlug(slug); err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &Region{
		ID: id, Slug: slug, Name: name, Status: RegionStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// NewGatewayDeployment builds a validated deployment entity (pending, with join token hash).
func NewGatewayDeployment(id, regionID, name, publicBaseURL, joinTokenHash string, now time.Time) (*GatewayDeployment, error) {
	name = strings.TrimSpace(name)
	publicBaseURL = strings.TrimSpace(publicBaseURL)
	if id == "" || regionID == "" {
		return nil, fmt.Errorf("%w: id and region_id required", kernel.ErrInvalidRequest)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	if joinTokenHash == "" {
		return nil, fmt.Errorf("%w: join_token_hash required", kernel.ErrInvalidRequest)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &GatewayDeployment{
		ID: id, RegionID: regionID, Name: name, PublicBaseURL: publicBaseURL,
		JoinTokenHash: joinTokenHash, Status: DeploymentStatusPending,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// EffectiveStatus returns the stored status, or stale when a healthy deployment
// has not heartbeated within ttl.
func (d GatewayDeployment) EffectiveStatus(now time.Time, ttl time.Duration) string {
	if d.Status == DeploymentStatusDisabled || d.Status == DeploymentStatusPending {
		return d.Status
	}
	if ttl <= 0 {
		ttl = DefaultHeartbeatTTL
	}
	if d.LastSeenAt == nil {
		if d.Status == DeploymentStatusHealthy {
			return DeploymentStatusStale
		}
		return d.Status
	}
	if d.Status == DeploymentStatusHealthy || d.Status == DeploymentStatusStale {
		if now.Sub(d.LastSeenAt.UTC()) > ttl {
			return DeploymentStatusStale
		}
		return DeploymentStatusHealthy
	}
	return d.Status
}
