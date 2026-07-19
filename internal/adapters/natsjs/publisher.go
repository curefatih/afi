package natsjs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/app/platform"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher publishes platform events to a NATS JetStream stream.
type Publisher struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	prefix string
}

type Config struct {
	URL           string
	Stream        string
	SubjectPrefix string
}

// Connect creates a JetStream publisher and ensures the stream exists.
func Connect(cfg Config) (*Publisher, error) {
	if cfg.URL == "" {
		cfg.URL = nats.DefaultURL
	}
	if cfg.Stream == "" {
		cfg.Stream = "AFI_PLATFORM"
	}
	if cfg.SubjectPrefix == "" {
		cfg.SubjectPrefix = "afi.platform"
	}
	nc, err := nats.Connect(cfg.URL,
		nats.Name("afi-platform-events"),
		nats.Timeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	subjects := []string{cfg.SubjectPrefix + ".>"}
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:      cfg.Stream,
		Subjects:  subjects,
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    7 * 24 * time.Hour,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure stream %s: %w", cfg.Stream, err)
	}
	return &Publisher{nc: nc, js: js, prefix: cfg.SubjectPrefix}, nil
}

func (p *Publisher) Publish(ctx context.Context, e platform.Event) error {
	if p == nil || p.js == nil {
		return fmt.Errorf("nats publisher not connected")
	}
	subject := p.prefix + "." + strings.ReplaceAll(string(e.Name), "/", ".")
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: subject,
		Data:    body,
		Header:  nats.Header{},
	}
	msg.Header.Set("Nats-Msg-Id", e.ID)
	msg.Header.Set("afi-event-name", string(e.Name))
	if e.OrganizationID != "" {
		msg.Header.Set("afi-organization-id", e.OrganizationID)
	}
	_, err = p.js.PublishMsg(ctx, msg)
	return err
}

func (p *Publisher) Close() {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
}
