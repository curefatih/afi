package tagquota

import (
	"context"
	"sync"
	"testing"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

type memCounters struct {
	mu   sync.Mutex
	used map[string]int64
}

func (m *memCounters) key(scopeType, scopeID, metric, window string) string {
	return scopeType + "|" + scopeID + "|" + metric + "|" + window
}

func (m *memCounters) Get(_ context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.used[m.key(scopeType, scopeID, metric, window)], nil
}

func (m *memCounters) Incr(_ context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := m.key(scopeType, scopeID, metric, window)
	m.used[k] += delta
	return m.used[k], nil
}

func TestTagQuotaIndependentUsers(t *testing.T) {
	t.Parallel()
	counters := &memCounters{used: map[string]int64{}}
	h := New(counters, Config{TagKey: "end-user-id", Limit: 2, Window: "total", ScopeParent: "project"})

	call := &sdkhook.CallContext{
		Principal: sdkhook.Principal{OrganizationID: "o1", ProjectID: "p1", APIKeyID: "k1"},
		Tags:      map[string]string{"end-user-id": "fatih"},
		Metadata:  map[string]any{},
	}
	for i := 0; i < 2; i++ {
		d, err := h.BeforeCall(context.Background(), call)
		if err != nil || !d.Allow {
			t.Fatalf("request %d: %+v err=%v", i, d, err)
		}
	}
	d, err := h.BeforeCall(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	if d.Allow {
		t.Fatal("expected deny after limit")
	}

	other := &sdkhook.CallContext{
		Principal: call.Principal,
		Tags:      map[string]string{"end-user-id": "alice"},
		Metadata:  map[string]any{},
	}
	d, err = h.BeforeCall(context.Background(), other)
	if err != nil || !d.Allow {
		t.Fatalf("other user should be allowed: %+v err=%v", d, err)
	}
}

func TestTagQuotaConcurrentRespectsLimit(t *testing.T) {
	t.Parallel()
	counters := &memCounters{used: map[string]int64{}}
	h := New(counters, Config{TagKey: "end-user-id", Limit: 10, Window: "total", ScopeParent: "project"})
	call := &sdkhook.CallContext{
		Principal: sdkhook.Principal{OrganizationID: "o1", ProjectID: "p1", APIKeyID: "k1"},
		Tags:      map[string]string{"end-user-id": "fatih"},
		Metadata:  map[string]any{},
	}

	const n = 50
	var (
		mu     sync.Mutex
		allowed int
		wg     sync.WaitGroup
	)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			d, err := h.BeforeCall(context.Background(), call)
			if err != nil {
				t.Errorf("BeforeCall: %v", err)
				return
			}
			if d.Allow {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 10 {
		t.Fatalf("allowed=%d want 10", allowed)
	}
	got, _ := counters.Get(context.Background(), "tag", "project:p1:end-user-id:fatih", "requests", "total")
	if got != 10 {
		t.Fatalf("counter=%d want 10", got)
	}
}

func TestTagQuotaSkipsMissingTag(t *testing.T) {
	t.Parallel()
	counters := &memCounters{used: map[string]int64{}}
	h := New(counters, Config{TagKey: "end-user-id", Limit: 1})
	call := &sdkhook.CallContext{
		Principal: sdkhook.Principal{OrganizationID: "o1", ProjectID: "p1"},
		Tags:      map[string]string{},
		Metadata:  map[string]any{},
	}
	d, err := h.BeforeCall(context.Background(), call)
	if err != nil || !d.Allow {
		t.Fatalf("%+v %v", d, err)
	}
	if len(counters.used) != 0 {
		t.Fatalf("expected no counter writes: %v", counters.used)
	}
}
