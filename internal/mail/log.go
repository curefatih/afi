package mail

import (
	"context"
	"log/slog"
)

// LogSender logs messages instead of delivering them (local/dev/tests).
type LogSender struct {
	Log *slog.Logger
}

func (s LogSender) Send(_ context.Context, msg Message) error {
	log := s.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("mail.send", "to", msg.To, "subject", msg.Subject)
	return nil
}
