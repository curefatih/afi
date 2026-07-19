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
	costs  []*float64
}

func (m *memUsage) InsertUsage(_ context.Context, e UsagePayload, costUSD *float64) error {
	m.events = append(m.events, e)
	m.costs = append(m.costs, costUSD)
	return nil
}

type memPrices struct {
	in, out float64
	ok      bool
}

func (m *memPrices) LookupModelPrice(context.Context, string, string) (float64, float64, bool, error) {
	return m.in, m.out, m.ok, nil
}

func TestProcessOutboxOnce(t *testing.T) {
	t.Parallel()
	payload, _ := json.Marshal(UsagePayload{
		OrganizationID: "o1", ProjectID: "p1", Model: "gpt-4o-mini",
		ProviderType: "openai", TargetModel: "gpt-4o-mini",
		Status: "ok", PromptTokens: 1_000_000, CompletionTokens: 0,
	})
	box := &memOutbox{rows: []OutboxRow{{ID: 1, Payload: payload}}}
	usage := &memUsage{}
	prices := &memPrices{in: 0.15, out: 0.60, ok: true}
	n, err := ProcessOnce(context.Background(), box, usage, prices)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(usage.events) != 1 {
		t.Fatalf("n=%d events=%+v", n, usage.events)
	}
	if box.rows[0].ProcessedAt == nil {
		t.Fatal("expected processed")
	}
	if usage.costs[0] == nil || *usage.costs[0] < 0.149 || *usage.costs[0] > 0.151 {
		t.Fatalf("cost=%v", usage.costs[0])
	}
}
