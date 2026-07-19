package mail

import (
	"context"
	"sync"
)

// MemorySender records sent messages for tests.
type MemorySender struct {
	mu   sync.Mutex
	Sent []Message
}

func (s *MemorySender) Send(_ context.Context, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sent = append(s.Sent, msg)
	return nil
}
