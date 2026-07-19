package platform

import (
	"context"
	"encoding/json"
	"testing"
)

type memEnq struct {
	payloads [][]byte
}

func (m *memEnq) Enqueue(_ context.Context, payload []byte) error {
	m.payloads = append(m.payloads, append([]byte(nil), payload...))
	return nil
}

func TestOutboxHandlerEnqueuesJSON(t *testing.T) {
	t.Parallel()
	enq := &memEnq{}
	bus := NewBus()
	bus.SubscribeAll(OutboxHandler(enq, nil))
	bus.Record(context.Background(), Event{
		Name: EventQuotaCreated, ResourceID: "q1", OrganizationID: "org",
	})
	if len(enq.payloads) != 1 {
		t.Fatalf("payloads=%d", len(enq.payloads))
	}
	var got Event
	if err := json.Unmarshal(enq.payloads[0], &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != EventQuotaCreated || got.ResourceID != "q1" || got.ID == "" {
		t.Fatalf("%+v", got)
	}
}
