package regions

import (
	"context"
	"time"
)

// Repository persists regions and gateway deployments.
type Repository interface {
	CreateRegion(ctx context.Context, r Region) error
	GetRegion(ctx context.Context, regionID string) (*Region, error)
	GetRegionBySlug(ctx context.Context, slug string) (*Region, error)
	ListRegions(ctx context.Context) ([]Region, error)
	UpdateRegion(ctx context.Context, regionID, name, status string) (*Region, error)
	CountDeployments(ctx context.Context, regionID string) (int, error)

	InsertDeployment(ctx context.Context, d GatewayDeployment) error
	GetDeployment(ctx context.Context, deploymentID string) (*GatewayDeployment, error)
	ListDeploymentsByRegion(ctx context.Context, regionID string) ([]GatewayDeployment, error)
	UpdateJoinTokenHash(ctx context.Context, deploymentID, joinTokenHash string) error
	RecordHeartbeat(ctx context.Context, deploymentID string, snapVersion int64, build string, at time.Time) error
	GetDeploymentByJoinTokenHash(ctx context.Context, tokenHash string) (*GatewayDeployment, error)
	UpdateDeploymentStatus(ctx context.Context, deploymentID, status string) error

	ListMembershipsByRegion(ctx context.Context, regionID string) ([]OrgRegionMembership, error)
	ListMembershipsByOrg(ctx context.Context, orgID string) ([]OrgRegionMembership, error)
	GetMembership(ctx context.Context, regionID, orgID string) (*OrgRegionMembership, error)
	UpsertMembership(ctx context.Context, m OrgRegionMembership) error
	DeleteMembership(ctx context.Context, regionID, orgID string) error

	GetOverlay(ctx context.Context, regionID, orgID string) (*RegionConfigOverlay, error)
	ListOverlaysByRegion(ctx context.Context, regionID string) ([]RegionConfigOverlay, error)
	UpsertOverlay(ctx context.Context, o RegionConfigOverlay) error
	DeleteOverlay(ctx context.Context, regionID, orgID string) error
}
