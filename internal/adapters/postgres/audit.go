package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/audit"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEvents implements audit.Store with Postgres.
type AuditEvents struct {
	Pool *pgxpool.Pool
}

var _ audit.Store = (*AuditEvents)(nil)

// Insert writes one audit entry. Empty organization_id is skipped (system-wide events).
func (a *AuditEvents) Insert(ctx context.Context, e audit.Entry) error {
	if a == nil || a.Pool == nil {
		return nil
	}
	if e.OrganizationID == "" {
		return nil
	}
	if e.ID == "" {
		return fmt.Errorf("audit event id required")
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if e.Summary == "" {
		e.Summary = audit.Summary(e.Name, e.ResourceID)
	}
	meta := e.Meta
	if meta == nil {
		meta = map[string]string{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = a.Pool.Exec(ctx, `
		INSERT INTO audit_events (id, name, organization_id, resource_id, actor_user_id, summary, meta, at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, e.ID, e.Name, e.OrganizationID, e.ResourceID, e.ActorUserID, e.Summary, metaJSON, e.At)
	return err
}

// List returns recent audit events for an organization.
func (a *AuditEvents) List(ctx context.Context, orgID string, f audit.Filter) ([]audit.Record, error) {
	if a == nil || a.Pool == nil {
		return nil, nil
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	args := []any{orgID}
	where := `e.organization_id = $1`
	n := 2
	if f.Name != "" {
		where += fmt.Sprintf(` AND e.name = $%d`, n)
		args = append(args, f.Name)
		n++
	}
	if f.From != nil {
		where += fmt.Sprintf(` AND e.at >= $%d`, n)
		args = append(args, *f.From)
		n++
	}
	if f.To != nil {
		where += fmt.Sprintf(` AND e.at <= $%d`, n)
		args = append(args, *f.To)
		n++
	}
	args = append(args, limit)
	q := fmt.Sprintf(`
		SELECT e.id, e.name, e.organization_id, e.resource_id, e.actor_user_id,
			COALESCE(u.email, ''), COALESCE(u.name, ''), e.summary, e.meta, e.at
		FROM audit_events e
		LEFT JOIN users u ON u.id = e.actor_user_id
		WHERE %s
		ORDER BY e.at DESC
		LIMIT $%d
	`, where, n)
	rows, err := a.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []audit.Record
	for rows.Next() {
		var r audit.Record
		var metaJSON []byte
		if err := rows.Scan(
			&r.ID, &r.Name, &r.OrganizationID, &r.ResourceID, &r.ActorUserID,
			&r.ActorEmail, &r.ActorName, &r.Summary, &metaJSON, &r.At,
		); err != nil {
			return nil, err
		}
		// Always derive from the current formatter so historical rows pick up
		// new event labels (summary is deterministic from name + resource_id).
		r.Summary = audit.Summary(r.Name, r.ResourceID)
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &r.Meta)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
