package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// EventName identifies a platform domain event.
type EventName string

const (
	EventOrgCreated            EventName = "org.created"
	EventMemberAdded           EventName = "member.added"
	EventMemberRoleUpdated     EventName = "member.role_updated"
	EventInviteCreated         EventName = "invite.created"
	EventInviteRevoked         EventName = "invite.revoked"
	EventInviteResent          EventName = "invite.resent"
	EventInviteAccepted        EventName = "invite.accepted"
	EventTeamCreated           EventName = "team.created"
	EventTeamMemberAdded       EventName = "team.member_added"
	EventTeamMemberRoleUpdated EventName = "team.member_role_updated"
	EventTeamMemberRemoved     EventName = "team.member_removed"
	EventProjectCreated        EventName = "project.created"
	EventAPIKeyCreated         EventName = "api_key.created"
	EventAPIKeyDeleted         EventName = "api_key.deleted"
	EventProviderCreated       EventName = "provider.created"
	EventProviderUpdated       EventName = "provider.updated"
	EventProviderDeleted       EventName = "provider.deleted"
	EventRouteCreated          EventName = "route.created"
	EventRouteUpdated          EventName = "route.updated"
	EventRouteDeleted          EventName = "route.deleted"
	EventOrgDefaultRetryUpdated EventName = "org.default_retry.updated"
	EventQuotaCreated          EventName = "quota.created"
	EventQuotaUpdated          EventName = "quota.updated"
	EventQuotaDeleted          EventName = "quota.deleted"
	EventPolicyCreated         EventName = "policy.created"
	EventPolicyUpdated         EventName = "policy.updated"
	EventPolicyDeleted         EventName = "policy.deleted"
	EventWasmHookCreated       EventName = "wasm_hook.created"
	EventWasmHookUpdated       EventName = "wasm_hook.updated"
	EventWasmHookDeleted       EventName = "wasm_hook.deleted"
	EventCredentialCreated     EventName = "credential.created"
	EventCredentialUpdated     EventName = "credential.updated"
	EventCredentialRotated     EventName = "credential.rotated"
	EventCredentialDeleted     EventName = "credential.deleted"
	EventCredentialAssigned    EventName = "credential.assigned"
	EventCredentialUnassigned  EventName = "credential.unassigned"
	EventSnapshotPublish       EventName = "snapshot.published"
)

// EventAll matches every event when used with Bus.Subscribe.
const EventAll EventName = "*"

// Event is a domain event emitted after successful platform work.
type Event struct {
	ID             string            `json:"id"`
	Name           EventName         `json:"name"`
	ResourceID     string            `json:"resource_id,omitempty"`
	OrganizationID string            `json:"organization_id,omitempty"`
	At             time.Time         `json:"at"`
	Meta           map[string]string `json:"meta,omitempty"`
}

// Handler receives a domain event. Handlers must not panic; the bus recovers panics.
type Handler func(ctx context.Context, e Event)

// EventRecorder receives domain events after successful platform commands.
type EventRecorder interface {
	Record(ctx context.Context, e Event)
}

// NopRecorder discards events.
type NopRecorder struct{}

func (NopRecorder) Record(context.Context, Event) {}

// MultiRecorder fans events out to multiple recorders.
type MultiRecorder []EventRecorder

func (m MultiRecorder) Record(ctx context.Context, e Event) {
	for _, r := range m {
		if r == nil {
			continue
		}
		r.Record(ctx, e)
	}
}

// Bus is an in-process pub/sub domain event bus.
// It implements EventRecorder and fans each event to matching subscribers.
type Bus struct {
	mu      sync.RWMutex
	subs    map[EventName][]Handler
	onPanic func(recovered any, e Event)
}

// NewBus creates an empty event bus.
func NewBus() *Bus {
	return &Bus{subs: make(map[EventName][]Handler)}
}

// Subscribe registers a handler for a specific event name.
// Use EventAll to receive every event.
func (b *Bus) Subscribe(name EventName, h Handler) {
	if b == nil || h == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[name] = append(b.subs[name], h)
}

// SubscribeAll is shorthand for Subscribe(EventAll, h).
func (b *Bus) SubscribeAll(h Handler) {
	b.Subscribe(EventAll, h)
}

// Record publishes an event to matching subscribers (specific name + EventAll).
// Subscriber panics are recovered so one bad handler cannot break the bus.
func (b *Bus) Record(ctx context.Context, e Event) {
	if b == nil {
		return
	}
	if e.ID == "" {
		e.ID = newEventID()
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	b.mu.RLock()
	handlers := append([]Handler{}, b.subs[e.Name]...)
	handlers = append(handlers, b.subs[EventAll]...)
	onPanic := b.onPanic
	b.mu.RUnlock()
	for _, h := range handlers {
		func(h Handler) {
			defer func() {
				if rec := recover(); rec != nil && onPanic != nil {
					onPanic(rec, e)
				}
			}()
			h(ctx, e)
		}(h)
	}
}

// OnPanic sets an optional callback when a subscriber panics.
func (b *Bus) OnPanic(fn func(recovered any, e Event)) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.onPanic = fn
	b.mu.Unlock()
}

// SlogHandler returns a Handler that logs events at debug level.
func SlogHandler(log *slog.Logger) Handler {
	return func(_ context.Context, e Event) {
		if log == nil {
			return
		}
		attrs := []any{
			"event_id", e.ID,
			"name", string(e.Name),
			"resource_id", e.ResourceID,
			"organization_id", e.OrganizationID,
			"at", e.At,
		}
		for k, v := range e.Meta {
			attrs = append(attrs, "meta_"+k, v)
		}
		log.Debug("platform event", attrs...)
	}
}

// MemoryRecorder stores recent events (newest last). Useful in tests and diagnostics.
type MemoryRecorder struct {
	mu    sync.Mutex
	limit int
	items []Event
}

// NewMemoryRecorder keeps up to limit events (default 256).
func NewMemoryRecorder(limit int) *MemoryRecorder {
	if limit <= 0 {
		limit = 256
	}
	return &MemoryRecorder{limit: limit}
}

func (m *MemoryRecorder) Record(_ context.Context, e Event) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, e)
	if len(m.items) > m.limit {
		m.items = append([]Event(nil), m.items[len(m.items)-m.limit:]...)
	}
}

// Snapshot returns a copy of recorded events (oldest first).
func (m *MemoryRecorder) Snapshot() []Event {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Event, len(m.items))
	copy(out, m.items)
	return out
}

func (s *Service) emit(ctx context.Context, name EventName, resourceID, orgID string) {
	if s.Events == nil {
		return
	}
	s.Events.Record(ctx, Event{
		ID:             newEventID(),
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

func newEventID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	return "evt_" + hex.EncodeToString(b[:])
}
