package memory

import (
	"sync"
	"time"

	"github.com/curefatih/afi/internal/identity"
)

// SSOStateStore is an in-memory CSRF state store for SSO (single-node).
type SSOStateStore struct {
	mu   sync.Mutex
	ttl  time.Duration
	data map[string]identity.SSOState
}

func NewSSOStateStore(ttl time.Duration) *SSOStateStore {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &SSOStateStore{
		ttl:  ttl,
		data: make(map[string]identity.SSOState),
	}
}

func (s *SSOStateStore) Put(state string, value identity.SSOState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeLocked(time.Now().UTC())
	if value.ExpiresAt.IsZero() {
		value.ExpiresAt = time.Now().UTC().Add(s.ttl)
	}
	s.data[state] = value
	return nil
}

func (s *SSOStateStore) Take(state string) (identity.SSOState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.purgeLocked(now)
	value, ok := s.data[state]
	if !ok {
		return identity.SSOState{}, false
	}
	delete(s.data, state)
	if !value.ExpiresAt.IsZero() && now.After(value.ExpiresAt) {
		return identity.SSOState{}, false
	}
	return value, true
}

func (s *SSOStateStore) purgeLocked(now time.Time) {
	for k, v := range s.data {
		if !v.ExpiresAt.IsZero() && now.After(v.ExpiresAt) {
			delete(s.data, k)
		}
	}
}

var _ identity.SSOStateStore = (*SSOStateStore)(nil)
