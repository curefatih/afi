package logpub

import (
	"context"
	"log/slog"

	"github.com/curefatih/afi/internal/app/platform"
)

// Publisher logs platform events (local/dev stand-in for a broker).
type Publisher struct {
	Log *slog.Logger
}

func (p Publisher) Publish(_ context.Context, e platform.Event) error {
	log := p.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("platform event published",
		"event_id", e.ID,
		"name", string(e.Name),
		"resource_id", e.ResourceID,
		"organization_id", e.OrganizationID,
		"at", e.At,
	)
	return nil
}
