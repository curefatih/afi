package platform

import (
	"context"
	"fmt"
	"time"
)

// EventName identifies a platform domain event.
type EventName string

const (
	EventProjectCreated  EventName = "project.created"
	EventAPIKeyCreated   EventName = "api_key.created"
	EventAPIKeyDeleted   EventName = "api_key.deleted"
	EventProviderCreated EventName = "provider.created"
	EventProviderUpdated EventName = "provider.updated"
	EventProviderDeleted EventName = "provider.deleted"
	EventRouteCreated    EventName = "route.created"
	EventRouteUpdated    EventName = "route.updated"
	EventRouteDeleted    EventName = "route.deleted"
	EventQuotaCreated    EventName = "quota.created"
	EventQuotaUpdated    EventName = "quota.updated"
	EventQuotaDeleted    EventName = "quota.deleted"
	EventPolicyCreated   EventName = "policy.created"
	EventPolicyUpdated   EventName = "policy.updated"
	EventPolicyDeleted   EventName = "policy.deleted"
	EventSnapshotPublish EventName = "snapshot.published"
)

// Event is a lightweight domain event emitted after successful platform work.
type Event struct {
	Name           EventName
	ResourceID     string
	OrganizationID string
	At             time.Time
}

func (s *Service) emit(ctx context.Context, name EventName, resourceID, orgID string) {
	if s.Events == nil {
		return
	}
	s.Events.Record(ctx, Event{
		Name:           name,
		ResourceID:     resourceID,
		OrganizationID: orgID,
		At:             time.Now().UTC(),
	})
}

func (s *Service) publish(ctx context.Context, action string) error {
	if s.Snap == nil {
		return fmt.Errorf("%s but snapshot publisher unavailable", action)
	}
	if err := s.Snap.PublishSnapshot(ctx); err != nil {
		return fmt.Errorf("%s but snapshot publish failed: %w", action, err)
	}
	s.emit(ctx, EventSnapshotPublish, "", "")
	return nil
}
