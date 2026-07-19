package kernel

import (
	"log/slog"
	"os"
)

func NewLogger(component string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With("component", component)
}
