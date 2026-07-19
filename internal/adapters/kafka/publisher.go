package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/app/platform"
	kafkago "github.com/segmentio/kafka-go"
)

// Publisher publishes platform events to a Kafka topic.
type Publisher struct {
	writer *kafkago.Writer
	topic  string
}

type Config struct {
	Brokers string // comma-separated
	Topic   string
}

// Connect creates a Kafka publisher.
func Connect(cfg Config) (*Publisher, error) {
	brokers := splitCSV(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers required")
	}
	if cfg.Topic == "" {
		cfg.Topic = "afi.platform.events"
	}
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafkago.Hash{},
		RequiredAcks: kafkago.RequireOne,
		Async:        false,
		BatchTimeout: 10 * time.Millisecond,
	}
	return &Publisher{writer: w, topic: cfg.Topic}, nil
}

func (p *Publisher) Publish(ctx context.Context, e platform.Event) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka publisher not connected")
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	key := e.OrganizationID
	if key == "" {
		key = string(e.Name)
	}
	msg := kafkago.Message{
		Key:   []byte(key),
		Value: body,
		Headers: []kafkago.Header{
			{Key: "afi-event-id", Value: []byte(e.ID)},
			{Key: "afi-event-name", Value: []byte(e.Name)},
		},
		Time: e.At,
	}
	if e.OrganizationID != "" {
		msg.Headers = append(msg.Headers, kafkago.Header{
			Key: "afi-organization-id", Value: []byte(e.OrganizationID),
		})
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *Publisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
