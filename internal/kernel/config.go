package kernel

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DatabaseURL string `yaml:"database_url" env:"AFI_DATABASE_URL"`
	RedisURL    string `yaml:"redis_url" env:"AFI_REDIS_URL"`

	ControlPlane struct {
		Addr string `yaml:"addr" env:"AFI_CONTROLPLANE_ADDR" env-default:":8081"`
	} `yaml:"controlplane"`

	Gateway struct {
		Addr                    string        `yaml:"addr" env:"AFI_GATEWAY_ADDR" env-default:":8080"`
		SnapshotPollIntervalRaw string        `yaml:"snapshot_poll_interval" env:"AFI_SNAPSHOT_POLL_INTERVAL" env-default:"2s"`
		SnapshotPollInterval    time.Duration `yaml:"-"`
	} `yaml:"gateway"`

	Auth struct {
		JWTSecret     string        `yaml:"jwt_secret" env:"AFI_JWT_SECRET"`
		TokenTTLRaw   string        `yaml:"token_ttl" env:"AFI_TOKEN_TTL" env-default:"24h"`
		TokenTTL      time.Duration `yaml:"-"`
		InternalToken string        `yaml:"internal_token" env:"AFI_INTERNAL_TOKEN"`
	} `yaml:"auth"`

	Seed struct {
		VirtualAPIKey   string `yaml:"virtual_api_key"`
		AdminEmail      string `yaml:"admin_email"`
		AdminPassword   string `yaml:"admin_password"`
		AdminName       string `yaml:"admin_name"`
		OpenAIBaseURL   string `yaml:"openai_base_url"`
		OpenAIAPIKeyEnv string `yaml:"openai_api_key_env"`
		DefaultModel    string `yaml:"default_model"`
	} `yaml:"seed"`

	// Events configures durable platform domain-event delivery.
	Events struct {
		// OutboxEnabled enqueues platform.Bus events into platform_event_outbox.
		OutboxEnabled bool `yaml:"outbox_enabled" env:"AFI_EVENTS_OUTBOX_ENABLED"`
		// Publisher is log | nats | kafka | noop (worker drain target).
		Publisher string `yaml:"publisher" env:"AFI_EVENTS_PUBLISHER" env-default:"log"`
		NATS      struct {
			URL           string `yaml:"url" env:"AFI_EVENTS_NATS_URL"`
			Stream        string `yaml:"stream" env:"AFI_EVENTS_NATS_STREAM"`
			SubjectPrefix string `yaml:"subject_prefix" env:"AFI_EVENTS_NATS_SUBJECT_PREFIX"`
		} `yaml:"nats"`
		Kafka struct {
			Brokers string `yaml:"brokers" env:"AFI_EVENTS_KAFKA_BROKERS"`
			Topic   string `yaml:"topic" env:"AFI_EVENTS_KAFKA_TOPIC"`
		} `yaml:"kafka"`
	} `yaml:"events"`

	// Mail configures outbound email for org member invites.
	Mail struct {
		PublicAppURL    string `yaml:"public_app_url" env:"AFI_MAIL_PUBLIC_APP_URL"`
		From            string `yaml:"from" env:"AFI_MAIL_FROM"`
		DefaultProvider string `yaml:"default_provider" env:"AFI_MAIL_DEFAULT_PROVIDER"`
		SMTP            struct {
			Enabled  bool   `yaml:"enabled" env:"AFI_MAIL_SMTP_ENABLED"`
			Host     string `yaml:"host" env:"AFI_MAIL_SMTP_HOST"`
			Port     int    `yaml:"port" env:"AFI_MAIL_SMTP_PORT"`
			Username string `yaml:"username" env:"AFI_MAIL_SMTP_USERNAME"`
			Password string `yaml:"password" env:"AFI_MAIL_SMTP_PASSWORD"`
			TLS      bool   `yaml:"tls" env:"AFI_MAIL_SMTP_TLS"`
		} `yaml:"smtp"`
		Resend struct {
			Enabled bool   `yaml:"enabled" env:"AFI_MAIL_RESEND_ENABLED"`
			APIKey  string `yaml:"api_key" env:"AFI_MAIL_RESEND_API_KEY"`
		} `yaml:"resend"`
		SES struct {
			Enabled         bool   `yaml:"enabled" env:"AFI_MAIL_SES_ENABLED"`
			Region          string `yaml:"region" env:"AFI_MAIL_SES_REGION"`
			AccessKeyID     string `yaml:"access_key_id" env:"AFI_MAIL_SES_ACCESS_KEY_ID"`
			SecretAccessKey string `yaml:"secret_access_key" env:"AFI_MAIL_SES_SECRET_ACCESS_KEY"`
		} `yaml:"ses"`
	} `yaml:"mail"`
}

func LoadConfig() (*Config, error) {
	path := os.Getenv("AFI_CONFIG")
	if path == "" {
		path = "configs/dev.yaml"
	}

	var cfg Config
	if _, err := os.Stat(path); err == nil {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
	} else if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env config: %w", err)
	}

	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "postgres://afi:afi@localhost:5433/afi?sslmode=disable"
	}
	if cfg.RedisURL == "" {
		cfg.RedisURL = "redis://localhost:6379/0"
	}
	if cfg.ControlPlane.Addr == "" {
		cfg.ControlPlane.Addr = ":8081"
	}
	if cfg.Gateway.Addr == "" {
		cfg.Gateway.Addr = ":8080"
	}
	if cfg.Gateway.SnapshotPollIntervalRaw == "" {
		cfg.Gateway.SnapshotPollIntervalRaw = "2s"
	}
	d, err := time.ParseDuration(cfg.Gateway.SnapshotPollIntervalRaw)
	if err != nil {
		return nil, fmt.Errorf("snapshot_poll_interval: %w", err)
	}
	cfg.Gateway.SnapshotPollInterval = d

	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "afi-local-dev-jwt-secret-change-me"
	}
	if cfg.Auth.InternalToken == "" {
		cfg.Auth.InternalToken = "afi-local-internal-token"
	}
	if cfg.Auth.TokenTTLRaw == "" {
		cfg.Auth.TokenTTLRaw = "24h"
	}
	ttl, err := time.ParseDuration(cfg.Auth.TokenTTLRaw)
	if err != nil {
		return nil, fmt.Errorf("token_ttl: %w", err)
	}
	cfg.Auth.TokenTTL = ttl

	if cfg.Seed.VirtualAPIKey == "" {
		cfg.Seed.VirtualAPIKey = "sk-project-local-dev-token-12345"
	}
	if cfg.Seed.AdminEmail == "" {
		cfg.Seed.AdminEmail = "admin@afi.local"
	}
	if cfg.Seed.AdminPassword == "" {
		cfg.Seed.AdminPassword = "admin"
	}
	if cfg.Seed.AdminName == "" {
		cfg.Seed.AdminName = "Admin"
	}
	if cfg.Seed.OpenAIBaseURL == "" {
		cfg.Seed.OpenAIBaseURL = "https://api.openai.com/v1"
	}
	if cfg.Seed.OpenAIAPIKeyEnv == "" {
		cfg.Seed.OpenAIAPIKeyEnv = "OPENAI_API_KEY"
	}
	if cfg.Seed.DefaultModel == "" {
		cfg.Seed.DefaultModel = "gpt-4o-mini"
	}
	if cfg.Events.Publisher == "" {
		cfg.Events.Publisher = "log"
	}
	if cfg.Events.NATS.URL == "" {
		cfg.Events.NATS.URL = "nats://127.0.0.1:4222"
	}
	if cfg.Events.NATS.Stream == "" {
		cfg.Events.NATS.Stream = "AFI_PLATFORM"
	}
	if cfg.Events.NATS.SubjectPrefix == "" {
		cfg.Events.NATS.SubjectPrefix = "afi.platform"
	}
	if cfg.Events.Kafka.Brokers == "" {
		cfg.Events.Kafka.Brokers = "127.0.0.1:9092"
	}
	if cfg.Events.Kafka.Topic == "" {
		cfg.Events.Kafka.Topic = "afi.platform.events"
	}

	if cfg.Mail.PublicAppURL == "" {
		cfg.Mail.PublicAppURL = "http://localhost:3000"
	}
	if cfg.Mail.From == "" {
		cfg.Mail.From = "AFI <noreply@afi.local>"
	}
	if cfg.Mail.DefaultProvider == "" {
		cfg.Mail.DefaultProvider = "smtp"
	}
	if cfg.Mail.SMTP.Host == "" {
		cfg.Mail.SMTP.Host = "localhost"
	}
	if cfg.Mail.SMTP.Port == 0 {
		cfg.Mail.SMTP.Port = 1025
	}
	return &cfg, nil
}
