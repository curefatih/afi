package platform

import (
	"context"
	"log/slog"
	"time"

	"github.com/curefatih/afi/internal/audit"
)

// AuditHandler maps domain events onto the audit.Store port.
// It belongs in the application layer so adapters stay free of the platform Event type.
func AuditHandler(store audit.Store, log *slog.Logger) Handler {
	return func(ctx context.Context, e Event) {
		if store == nil {
			return
		}
		entry := audit.Entry{
			ID:             e.ID,
			Name:           string(e.Name),
			OrganizationID: e.OrganizationID,
			ResourceID:     e.ResourceID,
			ActorUserID:    e.ActorUserID,
			Summary:        audit.Summary(string(e.Name), e.ResourceID),
			Meta:           e.Meta,
			At:             e.At,
		}
		if entry.At.IsZero() {
			entry.At = time.Now().UTC()
		}
		if err := store.Insert(ctx, entry); err != nil && log != nil {
			log.Error("audit insert", "err", err, "event_id", e.ID, "name", string(e.Name))
		}
	}
}
