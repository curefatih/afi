package kernel

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// SSOProvider is a platform-wide federated IdP configuration entry.
type SSOProvider struct {
	ID                   string   `yaml:"id"`
	Type                 string   `yaml:"type"` // oidc | oauth2 (saml reserved)
	DisplayName          string   `yaml:"display_name"`
	Issuer               string   `yaml:"issuer"`
	ClientID             string   `yaml:"client_id"`
	ClientSecret         string   `yaml:"client_secret"`
	Scopes               []string `yaml:"scopes"`
	AuthURL              string   `yaml:"auth_url"`
	TokenURL             string   `yaml:"token_url"`
	UserInfoURL          string   `yaml:"userinfo_url"`
	RequireEmailVerified *bool    `yaml:"require_email_verified"`
}

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
		// WasmBeforeCall is an optional path to a TinyGo .wasm exporting before_call.
		WasmBeforeCall string `yaml:"wasm_before_call" env:"AFI_WASM_BEFORE_CALL"`
		// WasmBeforeChat is an optional path to a TinyGo .wasm exporting before_chat.
		WasmBeforeChat string `yaml:"wasm_before_chat" env:"AFI_WASM_BEFORE_CHAT"`
		// WasmS3 is an optional S3-compatible store for module_uri values like s3://bucket/key.
		WasmS3 struct {
			Endpoint  string `yaml:"endpoint" env:"AFI_WASM_S3_ENDPOINT"`
			AccessKey string `yaml:"access_key" env:"AFI_WASM_S3_ACCESS_KEY"`
			SecretKey string `yaml:"secret_key" env:"AFI_WASM_S3_SECRET_KEY"`
			Region    string `yaml:"region" env:"AFI_WASM_S3_REGION"`
			UseSSL    bool   `yaml:"use_ssl" env:"AFI_WASM_S3_USE_SSL"`
			PathStyle bool   `yaml:"path_style" env:"AFI_WASM_S3_PATH_STYLE"`
		} `yaml:"wasm_s3"`
	} `yaml:"gateway"`

	Auth struct {
		JWTSecret     string        `yaml:"jwt_secret" env:"AFI_JWT_SECRET"`
		TokenTTLRaw   string        `yaml:"token_ttl" env:"AFI_TOKEN_TTL" env-default:"24h"`
		TokenTTL      time.Duration `yaml:"-"`
		InternalToken string        `yaml:"internal_token" env:"AFI_INTERNAL_TOKEN"`
		// PublicBaseURL is the externally reachable control-plane URL (SSO callbacks).
		PublicBaseURL string `yaml:"public_base_url" env:"AFI_AUTH_PUBLIC_BASE_URL"`
		SSO           struct {
			Enabled bool `yaml:"enabled" env:"AFI_SSO_ENABLED"`
			// StateStore is redis (shared, multi-node) or memory (single-node / tests).
			// Default redis — control plane is expected to scale horizontally.
			StateStore string        `yaml:"state_store" env:"AFI_SSO_STATE_STORE" env-default:"redis"`
			Providers  []SSOProvider `yaml:"providers"`
		} `yaml:"sso"`
	} `yaml:"auth"`

	// Credentials configures encryption for storage_kind=encrypted_db provider credentials.
	// Any non-empty string is accepted (SHA-256 derived); or base64:/raw 32-byte base64.
	Credentials struct {
		MasterKey string `yaml:"master_key" env:"AFI_CREDENTIALS_MASTER_KEY"`
	} `yaml:"credentials"`

	// Secrets configures external vault backends for storage_kind=vault credentials.
	Secrets struct {
		AWSSM struct {
			Enabled bool   `yaml:"enabled" env:"AFI_SECRETS_AWS_SM_ENABLED"`
			Region  string `yaml:"region" env:"AFI_SECRETS_AWS_SM_REGION"`
		} `yaml:"aws_sm"`
		Vault struct {
			Addr  string `yaml:"addr" env:"AFI_SECRETS_VAULT_ADDR"`
			Token string `yaml:"token" env:"AFI_SECRETS_VAULT_TOKEN"`
		} `yaml:"vault"`
	} `yaml:"secrets"`

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

	// Telemetry configures OpenTelemetry metrics and traces (OTLP + optional Prometheus).
	Telemetry struct {
		Enabled           bool    `yaml:"enabled" env:"AFI_TELEMETRY_ENABLED"`
		ServiceName       string  `yaml:"service_name" env:"AFI_TELEMETRY_SERVICE_NAME"`
		Environment       string  `yaml:"environment" env:"AFI_TELEMETRY_ENVIRONMENT"`
		OTLPEndpoint      string  `yaml:"otlp_endpoint" env:"AFI_TELEMETRY_OTLP_ENDPOINT"`
		OTLPProtocol      string  `yaml:"otlp_protocol" env:"AFI_TELEMETRY_OTLP_PROTOCOL" env-default:"http"`
		OTLPHeaders       string  `yaml:"otlp_headers" env:"AFI_TELEMETRY_OTLP_HEADERS"`
		OTLPInsecure      bool    `yaml:"otlp_insecure" env:"AFI_TELEMETRY_OTLP_INSECURE"`
		MetricsPrometheus bool    `yaml:"metrics_prometheus" env:"AFI_TELEMETRY_METRICS_PROMETHEUS"`
		TracesSampler     string  `yaml:"traces_sampler" env:"AFI_TELEMETRY_TRACES_SAMPLER" env-default:"parentbased_always_on"`
		TracesSamplerArg  float64 `yaml:"traces_sampler_arg" env:"AFI_TELEMETRY_TRACES_SAMPLER_ARG" env-default:"1.0"`
	} `yaml:"telemetry"`

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

	if cfg.Auth.PublicBaseURL == "" {
		cfg.Auth.PublicBaseURL = "http://localhost:8081"
	}
	if cfg.Auth.SSO.StateStore == "" {
		cfg.Auth.SSO.StateStore = "redis"
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Auth.SSO.StateStore)) {
	case "redis", "memory":
		cfg.Auth.SSO.StateStore = strings.ToLower(strings.TrimSpace(cfg.Auth.SSO.StateStore))
	default:
		return nil, fmt.Errorf("auth.sso.state_store: must be redis or memory, got %q", cfg.Auth.SSO.StateStore)
	}
	for i := range cfg.Auth.SSO.Providers {
		p := &cfg.Auth.SSO.Providers[i]
		if p.Type == "" {
			if p.Issuer != "" {
				p.Type = "oidc"
			} else {
				p.Type = "oauth2"
			}
		}
		if p.RequireEmailVerified == nil {
			v := strings.EqualFold(p.Type, "oidc")
			p.RequireEmailVerified = &v
		}
		if p.DisplayName == "" {
			p.DisplayName = p.ID
		}
	}

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

	if cfg.Telemetry.OTLPProtocol == "" {
		cfg.Telemetry.OTLPProtocol = "http"
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Telemetry.OTLPProtocol)) {
	case "http", "grpc":
		cfg.Telemetry.OTLPProtocol = strings.ToLower(strings.TrimSpace(cfg.Telemetry.OTLPProtocol))
	default:
		return nil, fmt.Errorf("telemetry.otlp_protocol: must be http or grpc, got %q", cfg.Telemetry.OTLPProtocol)
	}
	if cfg.Telemetry.TracesSampler == "" {
		cfg.Telemetry.TracesSampler = "parentbased_always_on"
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Telemetry.TracesSampler)) {
	case "parentbased_always_on", "parentbased_traceidratio":
		cfg.Telemetry.TracesSampler = strings.ToLower(strings.TrimSpace(cfg.Telemetry.TracesSampler))
	default:
		return nil, fmt.Errorf("telemetry.traces_sampler: must be parentbased_always_on or parentbased_traceidratio, got %q", cfg.Telemetry.TracesSampler)
	}
	if cfg.Telemetry.TracesSamplerArg <= 0 {
		cfg.Telemetry.TracesSamplerArg = 1.0
	}
	if cfg.Telemetry.TracesSamplerArg > 1 {
		cfg.Telemetry.TracesSamplerArg = 1.0
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
