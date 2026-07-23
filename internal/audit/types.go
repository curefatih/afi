// Package audit is the durable org-scoped audit trail bounded context.
// Persistence technology lives in adapters; this package has no DB/HTTP imports.
package audit

import (
	"context"
	"time"
)

// Entry is the write model persisted after a successful platform mutation.
type Entry struct {
	ID             string
	Name           string
	OrganizationID string
	ResourceID     string
	ActorUserID    string
	Summary        string
	Meta           map[string]string
	At             time.Time
}

// Record is the read model returned by list queries (may include actor join fields).
type Record struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	OrganizationID string            `json:"organization_id"`
	ResourceID     string            `json:"resource_id"`
	ActorUserID    string            `json:"actor_user_id"`
	ActorEmail     string            `json:"actor_email,omitempty"`
	ActorName      string            `json:"actor_name,omitempty"`
	Summary        string            `json:"summary"`
	Meta           map[string]string `json:"meta,omitempty"`
	At             time.Time         `json:"at"`
}

// Filter selects audit rows for an organization.
type Filter struct {
	Limit int
	From  *time.Time
	To    *time.Time
	Name  string // exact event name, optional
}

// Store persists and queries audit entries (implemented by infrastructure adapters).
type Store interface {
	Insert(ctx context.Context, e Entry) error
	List(ctx context.Context, orgID string, f Filter) ([]Record, error)
}
