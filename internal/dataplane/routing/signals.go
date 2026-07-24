package routing

import (
	"strings"
	"sync"
)

const ewmaAlpha = 0.2

// MemorySignalStore keeps gateway-local EWMA latency and error rates in process memory.
// Multi-instance skew is accepted; no Redis/Postgres.
type MemorySignalStore struct {
	mu   sync.Mutex
	byKey map[string]*signalEntry
}

type signalEntry struct {
	latencyEWMA float64
	hasLatency  bool
	successes   int64
	failures    int64
}

// NewMemorySignalStore returns an empty in-process signal store.
func NewMemorySignalStore() *MemorySignalStore {
	return &MemorySignalStore{byKey: make(map[string]*signalEntry)}
}

func signalKey(providerID, targetModel string) string {
	return strings.TrimSpace(providerID) + "\x00" + strings.TrimSpace(targetModel)
}

func (s *MemorySignalStore) Observe(providerID, targetModel string, latencyMs int64, failed bool) {
	if s == nil {
		return
	}
	key := signalKey(providerID, targetModel)
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.byKey[key]
	if e == nil {
		e = &signalEntry{}
		s.byKey[key] = e
	}
	if latencyMs < 0 {
		latencyMs = 0
	}
	sample := float64(latencyMs)
	if !e.hasLatency {
		e.latencyEWMA = sample
		e.hasLatency = true
	} else {
		e.latencyEWMA = ewmaAlpha*sample + (1-ewmaAlpha)*e.latencyEWMA
	}
	if failed {
		e.failures++
	} else {
		e.successes++
	}
}

func (s *MemorySignalStore) LatencyEWMA(providerID, targetModel string) (ms float64, ok bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.byKey[signalKey(providerID, targetModel)]
	if e == nil || !e.hasLatency {
		return 0, false
	}
	return e.latencyEWMA, true
}

func (s *MemorySignalStore) ErrorRate(providerID, targetModel string) (rate float64, ok bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.byKey[signalKey(providerID, targetModel)]
	if e == nil {
		return 0, false
	}
	total := e.successes + e.failures
	if total == 0 {
		return 0, false
	}
	return float64(e.failures) / float64(total), true
}
