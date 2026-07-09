package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Provider struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type ServerConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type HooksConfig struct {
	// maybe individual timeout for each hook?
	TimeoutMS int        `yaml:"timeout_ms"`
	HookSpecs []HookSpec `yaml:"hook_specs"`
}
type HookSpec struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type TelemetryConfig struct {
	Enabled      bool   `yaml:"enabled"`
	ServiceName  string `yaml:"service_name"`
	OTLPEndpoint string `yaml:"otlp_endpoint"`
	Insecure     bool   `yaml:"insecure"`
}

type DatabaseConfig struct {
	URL           string `yaml:"url"`
	MigrationsDir string `yaml:"migrations_dir"`
	AutoMigrate   bool   `yaml:"auto_migrate"`
}

type Config struct {
	Server    ServerConfig        `yaml:"server"`
	Providers map[string]Provider `yaml:"providers"`
	Hooks     HooksConfig         `yaml:"hooks"`
	Telemetry TelemetryConfig     `yaml:"telemetry"`
	Database  DatabaseConfig      `yaml:"database"`
}

func Load(path string) (*Config, error) {
	LoadDotEnv(".env", ".env.local")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	expanded := os.ExpandEnv(string(data))
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
}

func (c *Config) validate() error {
	if len(c.Providers) == 0 {
		return fmt.Errorf("at least one provider must be configured")
	}
	for name, p := range c.Providers {
		if strings.TrimSpace(p.BaseURL) == "" {
			return fmt.Errorf("provider %q: base_url is required", name)
		}
	}
	return nil
}
