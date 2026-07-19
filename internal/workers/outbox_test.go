package workers

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type memOutbox struct {
	rows []OutboxRow
}

func (m *memOutbox) ClaimBatch(_ context.Context, limit int) ([]OutboxRow, error) {
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

func (m *memOutbox) MarkProcessed(_ context.Context, id int64) error {
	now := time.Now().UTC()
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].ProcessedAt = &now
		}
	}
	return nil
}

type memUsage struct {
	events []UsagePayload
}

func (m *memUsage) InsertUsage(_ context.Context, e UsagePayload) error {
	m.events = append(m.events, e)
	return nil
}

func TestProcessOutboxOnce(t *testing.T) {
	t.Parallel()
	payload, _ := json.Marshal(UsagePayload{
		OrganizationID: "o1", ProjectID: "p1", Model: "gpt-4o-mini", Status: "ok",
	})
	box := &memOutbox{rows: []OutboxRow{{ID: 1, Payload: payload}}}
	usage := &memUsage{}
	n, err := ProcessOnce(context.Background(), box, usage)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(usage.events) != 1 {
		t.Fatalf("n=%d events=%+v", n, usage.events)
	}
	if box.rows[0].ProcessedAt == nil {
		t.Fatal("expected processed")
	}
}
