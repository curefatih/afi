package dataplane

import (
	"sync"
	"sync/atomic"

	"github.com/curefatih/afi/internal/snapshot"
)

// Holder keeps the current immutable snapshot for the request path.
type Holder struct {
	v  atomic.Pointer[snapshot.Snapshot]
	mu sync.Mutex
}

func NewHolder() *Holder {
	return &Holder{}
}

func (h *Holder) Set(s *snapshot.Snapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.v.Store(s)
}

func (h *Holder) Get() *snapshot.Snapshot {
	return h.v.Load()
}
