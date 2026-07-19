package eventpub

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/curefatih/afi/internal/adapters/kafka"
	"github.com/curefatih/afi/internal/adapters/logpub"
	"github.com/curefatih/afi/internal/adapters/natsjs"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/workers"
)

// New builds a platform event publisher from config.
// Supported publishers: log (default), nats, kafka, noop.
func New(cfg *kernel.Config, log *slog.Logger) (workers.EventPublisher, func(), error) {
	switch cfg.Events.Publisher {
	case "", "log":
		return logpub.Publisher{Log: log}, func() {}, nil
	case "noop":
		return noopPublisher{}, func() {}, nil
	case "nats":
		p, err := natsjs.Connect(natsjs.Config{
			URL:           cfg.Events.NATS.URL,
			Stream:        cfg.Events.NATS.Stream,
			SubjectPrefix: cfg.Events.NATS.SubjectPrefix,
		})
		if err != nil {
			return nil, nil, err
		}
		return p, p.Close, nil
	case "kafka":
		p, err := kafka.Connect(kafka.Config{
			Brokers: cfg.Events.Kafka.Brokers,
			Topic:   cfg.Events.Kafka.Topic,
		})
		if err != nil {
			return nil, nil, err
		}
		return p, func() { _ = p.Close() }, nil
	default:
		return nil, nil, fmt.Errorf("unknown events.publisher %q (want log|nats|kafka|noop)", cfg.Events.Publisher)
	}
}

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, platform.Event) error { return nil }
