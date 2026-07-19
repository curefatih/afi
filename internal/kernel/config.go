package kernel

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DatabaseURL string `yaml:"database_url" env:"AFI_DATABASE_URL"`

	ControlPlane struct {
		Addr string `yaml:"addr" env:"AFI_CONTROLPLANE_ADDR" env-default:":8081"`
	} `yaml:"controlplane"`

	Gateway struct {
		Addr                   string `yaml:"addr" env:"AFI_GATEWAY_ADDR" env-default:":8080"`
		SnapshotPollIntervalRaw string `yaml:"snapshot_poll_interval" env:"AFI_SNAPSHOT_POLL_INTERVAL" env-default:"2s"`
		SnapshotPollInterval   time.Duration `yaml:"-"`
	} `yaml:"gateway"`

	Auth struct {
		JWTSecret   string        `yaml:"jwt_secret" env:"AFI_JWT_SECRET"`
		TokenTTLRaw string        `yaml:"token_ttl" env:"AFI_TOKEN_TTL" env-default:"24h"`
		TokenTTL    time.Duration `yaml:"-"`
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
}

func LoadConfig() (*Config, error) {
	path := os.Getenv("AFI_CONFIG")
	if path == "" {
		path = "configs/local.yaml"
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

	return &cfg, nil
}
