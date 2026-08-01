package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegionsStore implements regions.Repository.
type RegionsStore struct {
	Pool *pgxpool.Pool
}

func NewRegionsStore(pool *pgxpool.Pool) *RegionsStore {
	return &RegionsStore{Pool: pool}
}

func (s *RegionsStore) CreateRegion(ctx context.Context, r regions.Region) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO regions (id, slug, name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, r.ID, r.Slug, r.Name, r.Status, r.CreatedAt, r.UpdatedAt)
	return err
}

func (s *RegionsStore) GetRegion(ctx context.Context, regionID string) (*regions.Region, error) {
	var r regions.Region
	err := s.Pool.QueryRow(ctx, `
		SELECT id, slug, name, status, created_at, updated_at
		FROM regions WHERE id = $1
	`, regionID).Scan(&r.ID, &r.Slug, &r.Name, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *RegionsStore) GetRegionBySlug(ctx context.Context, slug string) (*regions.Region, error) {
	var r regions.Region
	err := s.Pool.QueryRow(ctx, `
		SELECT id, slug, name, status, created_at, updated_at
		FROM regions WHERE slug = $1
	`, slug).Scan(&r.ID, &r.Slug, &r.Name, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *RegionsStore) ListRegions(ctx context.Context) ([]regions.Region, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, slug, name, status, created_at, updated_at
		FROM regions ORDER BY slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []regions.Region
	for rows.Next() {
		var r regions.Region
		if err := rows.Scan(&r.ID, &r.Slug, &r.Name, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *RegionsStore) UpdateRegion(ctx context.Context, regionID, name, status string) (*regions.Region, error) {
	var r regions.Region
	err := s.Pool.QueryRow(ctx, `
		UPDATE regions SET name = $2, status = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, slug, name, status, created_at, updated_at
	`, regionID, name, status).Scan(&r.ID, &r.Slug, &r.Name, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *RegionsStore) CountDeployments(ctx context.Context, regionID string) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gateway_deployments WHERE region_id = $1`, regionID).Scan(&n)
	return n, err
}

func (s *RegionsStore) InsertDeployment(ctx context.Context, d regions.GatewayDeployment) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO gateway_deployments (
			id, region_id, name, public_base_url, join_token_hash, status,
			last_seen_at, reported_snapshot_version, reported_build, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, d.ID, d.RegionID, d.Name, d.PublicBaseURL, d.JoinTokenHash, d.Status,
		d.LastSeenAt, d.ReportedSnapshotVersion, d.ReportedBuild, d.CreatedAt, d.UpdatedAt)
	return err
}

func (s *RegionsStore) GetDeployment(ctx context.Context, deploymentID string) (*regions.GatewayDeployment, error) {
	return s.scanDeployment(ctx, `
		SELECT id, region_id, name, public_base_url, join_token_hash, status,
			last_seen_at, reported_snapshot_version, reported_build, created_at, updated_at
		FROM gateway_deployments WHERE id = $1
	`, deploymentID)
}

func (s *RegionsStore) ListDeploymentsByRegion(ctx context.Context, regionID string) ([]regions.GatewayDeployment, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, region_id, name, public_base_url, join_token_hash, status,
			last_seen_at, reported_snapshot_version, reported_build, created_at, updated_at
		FROM gateway_deployments WHERE region_id = $1 ORDER BY name
	`, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []regions.GatewayDeployment
	for rows.Next() {
		d, err := scanDeploymentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (s *RegionsStore) UpdateJoinTokenHash(ctx context.Context, deploymentID, joinTokenHash string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE gateway_deployments SET join_token_hash = $2, updated_at = NOW() WHERE id = $1
	`, deploymentID, joinTokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *RegionsStore) RecordHeartbeat(ctx context.Context, deploymentID string, snapVersion int64, build string, at time.Time) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE gateway_deployments
		SET status = $2, last_seen_at = $3, reported_snapshot_version = $4,
		    reported_build = $5, updated_at = $3
		WHERE id = $1
	`, deploymentID, regions.DeploymentStatusHealthy, at, snapVersion, build)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *RegionsStore) GetDeploymentByJoinTokenHash(ctx context.Context, tokenHash string) (*regions.GatewayDeployment, error) {
	return s.scanDeployment(ctx, `
		SELECT id, region_id, name, public_base_url, join_token_hash, status,
			last_seen_at, reported_snapshot_version, reported_build, created_at, updated_at
		FROM gateway_deployments WHERE join_token_hash = $1
	`, tokenHash)
}

func (s *RegionsStore) UpdateDeploymentStatus(ctx context.Context, deploymentID, status string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE gateway_deployments SET status = $2, updated_at = NOW() WHERE id = $1
	`, deploymentID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

type deploymentRow interface {
	Scan(dest ...any) error
}

func (s *RegionsStore) scanDeployment(ctx context.Context, query string, arg any) (*regions.GatewayDeployment, error) {
	row := s.Pool.QueryRow(ctx, query, arg)
	d, err := scanDeploymentRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return d, err
}

func scanDeploymentRow(row deploymentRow) (*regions.GatewayDeployment, error) {
	var d regions.GatewayDeployment
	err := row.Scan(
		&d.ID, &d.RegionID, &d.Name, &d.PublicBaseURL, &d.JoinTokenHash, &d.Status,
		&d.LastSeenAt, &d.ReportedSnapshotVersion, &d.ReportedBuild, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *RegionsStore) ListMembershipsByRegion(ctx context.Context, regionID string) ([]regions.OrgRegionMembership, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT organization_id, region_id, status, created_at, updated_at
		FROM org_region_memberships WHERE region_id = $1 ORDER BY organization_id
	`, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []regions.OrgRegionMembership
	for rows.Next() {
		var m regions.OrgRegionMembership
		if err := rows.Scan(&m.OrganizationID, &m.RegionID, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *RegionsStore) ListMembershipsByOrg(ctx context.Context, orgID string) ([]regions.OrgRegionMembership, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT organization_id, region_id, status, created_at, updated_at
		FROM org_region_memberships WHERE organization_id = $1 ORDER BY region_id
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []regions.OrgRegionMembership
	for rows.Next() {
		var m regions.OrgRegionMembership
		if err := rows.Scan(&m.OrganizationID, &m.RegionID, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *RegionsStore) GetMembership(ctx context.Context, regionID, orgID string) (*regions.OrgRegionMembership, error) {
	var m regions.OrgRegionMembership
	err := s.Pool.QueryRow(ctx, `
		SELECT organization_id, region_id, status, created_at, updated_at
		FROM org_region_memberships WHERE region_id = $1 AND organization_id = $2
	`, regionID, orgID).Scan(&m.OrganizationID, &m.RegionID, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *RegionsStore) UpsertMembership(ctx context.Context, m regions.OrgRegionMembership) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO org_region_memberships (organization_id, region_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id, region_id) DO UPDATE
		SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at
	`, m.OrganizationID, m.RegionID, m.Status, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *RegionsStore) DeleteMembership(ctx context.Context, regionID, orgID string) error {
	tag, err := s.Pool.Exec(ctx, `
		DELETE FROM org_region_memberships WHERE region_id = $1 AND organization_id = $2
	`, regionID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *RegionsStore) GetOverlay(ctx context.Context, regionID, orgID string) (*regions.RegionConfigOverlay, error) {
	var o regions.RegionConfigOverlay
	var payload []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT organization_id, region_id, payload, created_at, updated_at
		FROM region_config_overlays WHERE region_id = $1 AND organization_id = $2
	`, regionID, orgID).Scan(&o.OrganizationID, &o.RegionID, &payload, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p, err := regions.UnmarshalOverlayPayload(payload)
	if err != nil {
		return nil, err
	}
	o.Payload = p
	return &o, nil
}

func (s *RegionsStore) ListOverlaysByRegion(ctx context.Context, regionID string) ([]regions.RegionConfigOverlay, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT organization_id, region_id, payload, created_at, updated_at
		FROM region_config_overlays WHERE region_id = $1 ORDER BY organization_id
	`, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []regions.RegionConfigOverlay
	for rows.Next() {
		var o regions.RegionConfigOverlay
		var payload []byte
		if err := rows.Scan(&o.OrganizationID, &o.RegionID, &payload, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		p, err := regions.UnmarshalOverlayPayload(payload)
		if err != nil {
			return nil, err
		}
		o.Payload = p
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *RegionsStore) UpsertOverlay(ctx context.Context, o regions.RegionConfigOverlay) error {
	payload, err := o.Payload.MarshalPayload()
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO region_config_overlays (organization_id, region_id, payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id, region_id) DO UPDATE
		SET payload = EXCLUDED.payload, updated_at = EXCLUDED.updated_at
	`, o.OrganizationID, o.RegionID, payload, o.CreatedAt, o.UpdatedAt)
	return err
}

func (s *RegionsStore) DeleteOverlay(ctx context.Context, regionID, orgID string) error {
	_, err := s.Pool.Exec(ctx, `
		DELETE FROM region_config_overlays WHERE region_id = $1 AND organization_id = $2
	`, regionID, orgID)
	return err
}
