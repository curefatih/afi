package controlplane

import (
	"context"
	"log/slog"

	"github.com/curefatih/afi/internal/app/platform"
)

// slogEventRecorder logs platform domain events at debug level.
type slogEventRecorder struct {
	log *slog.Logger
}

func (r slogEventRecorder) Record(_ context.Context, e platform.Event) {
	if r.log == nil {
		return
	}
	r.log.Debug("platform event",
		"name", string(e.Name),
		"resource_id", e.ResourceID,
		"organization_id", e.OrganizationID,
		"at", e.At,
	)
}
