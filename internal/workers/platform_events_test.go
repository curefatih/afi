package workers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/app/platform"
)

type memEventOutbox struct {
	rows []OutboxRow
}

func (m *memEventOutbox) ClaimBatch(_ context.Context, limit int) ([]OutboxRow, error) {
	var out []OutboxRow
	for i := range m.rows {
		if m.rows[i].ProcessedAt == nil {
			out = append(out, m.rows[i])
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *memEventOutbox) MarkProcessed(_ context.Context, id int64) error {
	now := time.Now().UTC()
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].ProcessedAt = &now
		}
	}
	return nil
}

type memPublisher struct {
	events []platform.Event
}

func (m *memPublisher) Publish(_ context.Context, e platform.Event) error {
	m.events = append(m.events, e)
	return nil
}

func TestProcessPlatformEventsOnce(t *testing.T) {
	t.Parallel()
	payload, _ := json.Marshal(platform.Event{
		ID: "evt_1", Name: platform.EventQuotaCreated, ResourceID: "q1", OrganizationID: "org",
	})
	box := &memEventOutbox{rows: []OutboxRow{{ID: 1, Payload: payload}}}
	pub := &memPublisher{}
	n, err := ProcessPlatformEventsOnce(context.Background(), box, pub)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(pub.events) != 1 || pub.events[0].Name != platform.EventQuotaCreated {
		t.Fatalf("n=%d events=%+v", n, pub.events)
	}
	if box.rows[0].ProcessedAt == nil {
		t.Fatal("expected processed")
	}
}
