package platform

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestBusFanOutAndWildcard(t *testing.T) {
	t.Parallel()
	bus := NewBus()
	var specific, all atomic.Int32
	bus.Subscribe(EventQuotaCreated, func(context.Context, Event) { specific.Add(1) })
	bus.SubscribeAll(func(context.Context, Event) { all.Add(1) })

	bus.Record(context.Background(), Event{Name: EventQuotaCreated, ResourceID: "q1"})
	bus.Record(context.Background(), Event{Name: EventRouteCreated, ResourceID: "r1"})

	if specific.Load() != 1 {
		t.Fatalf("specific=%d", specific.Load())
	}
	if all.Load() != 2 {
		t.Fatalf("all=%d", all.Load())
	}
}

func TestBusRecoversSubscriberPanic(t *testing.T) {
	t.Parallel()
	bus := NewBus()
	var ok atomic.Int32
	bus.SubscribeAll(func(context.Context, Event) { panic("boom") })
	bus.SubscribeAll(func(context.Context, Event) { ok.Add(1) })
	bus.Record(context.Background(), Event{Name: EventAPIKeyDeleted})
	if ok.Load() != 1 {
		t.Fatalf("ok=%d", ok.Load())
	}
}

func TestMemoryRecorderRing(t *testing.T) {
	t.Parallel()
	mem := NewMemoryRecorder(2)
	mem.Record(context.Background(), Event{Name: EventQuotaCreated, ResourceID: "1"})
	mem.Record(context.Background(), Event{Name: EventQuotaUpdated, ResourceID: "2"})
	mem.Record(context.Background(), Event{Name: EventQuotaDeleted, ResourceID: "3"})
	got := mem.Snapshot()
	if len(got) != 2 || got[0].ResourceID != "2" || got[1].ResourceID != "3" {
		t.Fatalf("%+v", got)
	}
}

func TestMultiRecorder(t *testing.T) {
	t.Parallel()
	a := NewMemoryRecorder(10)
	b := NewMemoryRecorder(10)
	MultiRecorder{a, b, nil}.Record(context.Background(), Event{Name: EventPolicyCreated, ID: "x"})
	if len(a.Snapshot()) != 1 || len(b.Snapshot()) != 1 {
		t.Fatal("expected both recorders")
	}
}
