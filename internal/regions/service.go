package regions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

var timeNowUTC = func() time.Time { return time.Now().UTC() }

// CreateRegion creates a validated region.
func CreateRegion(ctx context.Context, repo Repository, id, slug, name string) (*Region, error) {
	r, err := NewRegion(id, slug, name, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if existing, err := repo.GetRegionBySlug(ctx, r.Slug); err == nil && existing != nil {
		return nil, fmt.Errorf("%w: region slug already exists", kernel.ErrInvalidRequest)
	} else if err != nil && !errors.Is(err, kernel.ErrNotFound) {
		return nil, err
	}
	if err := repo.CreateRegion(ctx, *r); err != nil {
		return nil, err
	}
	return r, nil
}

// UpdateRegion patches name and/or status.
func UpdateRegion(ctx context.Context, repo Repository, regionID, name, status string) (*Region, error) {
	cur, err := repo.GetRegion(ctx, regionID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	status = strings.TrimSpace(status)
	if name == "" {
		name = cur.Name
	}
	if status == "" {
		status = cur.Status
	}
	if err := ValidateRegionStatus(status); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	return repo.UpdateRegion(ctx, regionID, name, status)
}

// RegisterDeployment creates a deployment and returns the raw join token once.
func RegisterDeployment(ctx context.Context, repo Repository, id, regionID, name, publicBaseURL string) (*DeploymentWithToken, error) {
	region, err := repo.GetRegion(ctx, regionID)
	if err != nil {
		return nil, err
	}
	if region.Status == RegionStatusDisabled {
		return nil, fmt.Errorf("%w: region is disabled", kernel.ErrInvalidRequest)
	}
	raw, hash, err := identity.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	d, err := NewGatewayDeployment(id, regionID, name, publicBaseURL, hash, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.InsertDeployment(ctx, *d); err != nil {
		return nil, err
	}
	return &DeploymentWithToken{Deployment: *d, JoinToken: raw}, nil
}

// RotateJoinToken issues a new join token for a deployment.
func RotateJoinToken(ctx context.Context, repo Repository, deploymentID string) (*DeploymentWithToken, error) {
	d, err := repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	raw, hash, err := identity.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	if err := repo.UpdateJoinTokenHash(ctx, deploymentID, hash); err != nil {
		return nil, err
	}
	d.JoinTokenHash = hash
	d.UpdatedAt = timeNowUTC()
	return &DeploymentWithToken{Deployment: *d, JoinToken: raw}, nil
}

// RecordHeartbeat authenticates by join token hash and updates liveness.
func RecordHeartbeat(ctx context.Context, repo Repository, deploymentID, joinTokenHash string, snapVersion int64, build string) (*GatewayDeployment, error) {
	d, err := repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	if d.Status == DeploymentStatusDisabled {
		return nil, fmt.Errorf("%w: deployment is disabled", kernel.ErrUnauthorized)
	}
	if d.JoinTokenHash == "" || d.JoinTokenHash != joinTokenHash {
		return nil, kernel.ErrUnauthorized
	}
	now := timeNowUTC()
	if err := repo.RecordHeartbeat(ctx, deploymentID, snapVersion, strings.TrimSpace(build), now); err != nil {
		return nil, err
	}
	d.Status = DeploymentStatusHealthy
	d.LastSeenAt = &now
	d.ReportedSnapshotVersion = snapVersion
	d.ReportedBuild = strings.TrimSpace(build)
	d.UpdatedAt = now
	return d, nil
}

// AuthenticateJoinToken resolves a deployment by raw join token.
func AuthenticateJoinToken(ctx context.Context, repo Repository, rawToken string) (*GatewayDeployment, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, kernel.ErrUnauthorized
	}
	hash := identity.HashOpaqueToken(rawToken)
	d, err := repo.GetDeploymentByJoinTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil, kernel.ErrUnauthorized
		}
		return nil, err
	}
	if d.Status == DeploymentStatusDisabled {
		return nil, kernel.ErrUnauthorized
	}
	return d, nil
}

// ListDeployments returns deployments with EffectiveStatus applied.
func ListDeployments(ctx context.Context, repo Repository, regionID string, ttl time.Duration) ([]GatewayDeployment, error) {
	items, err := repo.ListDeploymentsByRegion(ctx, regionID)
	if err != nil {
		return nil, err
	}
	now := timeNowUTC()
	for i := range items {
		items[i].Status = items[i].EffectiveStatus(now, ttl)
	}
	return items, nil
}

// GetDeployment returns a deployment with EffectiveStatus applied.
func GetDeployment(ctx context.Context, repo Repository, deploymentID string, ttl time.Duration) (*GatewayDeployment, error) {
	d, err := repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	d.Status = d.EffectiveStatus(timeNowUTC(), ttl)
	return d, nil
}

// BindOrgToRegion creates or updates membership.
func BindOrgToRegion(ctx context.Context, repo Repository, regionID, orgID, status string) (*OrgRegionMembership, error) {
	region, err := repo.GetRegion(ctx, regionID)
	if err != nil {
		return nil, err
	}
	if region.Status == RegionStatusDisabled {
		return nil, fmt.Errorf("%w: region is disabled", kernel.ErrInvalidRequest)
	}
	m, err := NewOrgRegionMembership(orgID, regionID, status, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.UpsertMembership(ctx, *m); err != nil {
		return nil, err
	}
	return m, nil
}

// UnbindOrgFromRegion removes membership and any overlay.
func UnbindOrgFromRegion(ctx context.Context, repo Repository, regionID, orgID string) error {
	if _, err := repo.GetMembership(ctx, regionID, orgID); err != nil {
		return err
	}
	_ = repo.DeleteOverlay(ctx, regionID, orgID)
	return repo.DeleteMembership(ctx, regionID, orgID)
}

// PutOverlay upserts a full replacement overlay for a bound org.
func PutOverlay(ctx context.Context, repo Repository, regionID, orgID string, payload OverlayPayload) (*RegionConfigOverlay, error) {
	m, err := repo.GetMembership(ctx, regionID, orgID)
	if err != nil {
		return nil, err
	}
	if m.Status != MembershipStatusActive {
		return nil, fmt.Errorf("%w: organization is not actively bound to region", kernel.ErrInvalidRequest)
	}
	o, err := NewRegionConfigOverlay(orgID, regionID, payload, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.UpsertOverlay(ctx, *o); err != nil {
		return nil, err
	}
	return o, nil
}

// DeleteOverlay reverts the org to inherit base config in the region.
func DeleteOverlay(ctx context.Context, repo Repository, regionID, orgID string) error {
	return repo.DeleteOverlay(ctx, regionID, orgID)
}

// BindAllOrganizations binds every org ID to the region (active). Idempotent upserts.
// Returns the number of organizations successfully bound.
func BindAllOrganizations(ctx context.Context, repo Repository, regionID string, orgIDs []string) (int, error) {
	if _, err := repo.GetRegion(ctx, regionID); err != nil {
		return 0, err
	}
	n := 0
	for _, orgID := range orgIDs {
		orgID = strings.TrimSpace(orgID)
		if orgID == "" {
			continue
		}
		if _, err := BindOrgToRegion(ctx, repo, regionID, orgID, MembershipStatusActive); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
