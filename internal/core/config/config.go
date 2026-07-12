package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	AppEnvironment      string               `yaml:"environment" env:"APP_ENV" env-default:"local"`
	HTTP                HTTPConfig           `yaml:"http"`
	Database            DatabaseConfig       `yaml:"database"`
	Providers           ProvidersConfig      `yaml:"providers"`
	Hooks               HooksConfig          `yaml:"hooks"`
	Organizations       []Organization       `yaml:"organizations"`
	Teams               []Team               `yaml:"teams"`
	Projects            []Project            `yaml:"projects"`
	APIKeys             []APIKey             `yaml:"api_keys"`
	UpstreamCredentials []UpstreamCredential `yaml:"upstream_credentials"`
	Budgets             []Budget             `yaml:"budgets"`
	RoutingRules        []RoutingRule        `yaml:"routing_rules"`
	Auth                AuthConfig           `yaml:"auth"`
}

type AuthConfig struct {
	TokenSecret   string        `yaml:"token_secret"`
	TokenDuration time.Duration `yaml:"token_duration"`
	Issuer        string        `yaml:"issuer"`
}

type Organization struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type Team struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type Project struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type APIKey struct {
	RawKeyPrefix string `yaml:"raw_key_prefix"`
	Type         string `yaml:"type"`
	ProjectID    string `yaml:"project_id"`
}

type UpstreamCredential struct {
	ProjectID string `yaml:"project_id"`
	Provider  string `yaml:"provider"`
	EnvVarKey string `yaml:"env_var_key"`
}

type Budget struct {
	Scope    string  `yaml:"scope"`
	TargetID string  `yaml:"target_id"`
	MaxCost  float64 `yaml:"max_cost"`
	UsedCost float64 `yaml:"used_cost"`
}

type RoutingRule struct {
	ID         string      `yaml:"id"`
	Name       string      `yaml:"name"`
	Priority   int         `yaml:"priority"`
	IsActive   bool        `yaml:"is_active"`
	Conditions []Condition `yaml:"conditions"`
	Target     Target      `yaml:"target"`
}

type Condition struct {
	Key      string `yaml:"key"`
	Operator string `yaml:"operator"`
	Value    string `yaml:"value"`
}

type Target struct {
	Provider    string `yaml:"provider"`
	TargetModel string `yaml:"target_model"`
}

type HooksConfig struct {
	TimeoutMs int        `yaml:"timeout_ms" env:"HOOKS_TIMEOUT_MS" env-default:"50"`
	Specs     []HookSpec `yaml:"specs"`
}

type HookSpec struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type ProvidersConfig struct {
	OpenAI    OpenAIConfig    `yaml:"openai"`
	Anthropic AnthropicConfig `yaml:"anthropic"`
}

type OpenAIConfig struct {
	APIKey string `yaml:"api_key" env:"OPENAI_API_KEY"`
}

type AnthropicConfig struct {
	APIKey string `yaml:"api_key" env:"ANTHROPIC_API_KEY"`
}

type HTTPConfig struct {
	Port         int           `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"0s"`
}

type DatabaseConfig struct {
	Host             string `yaml:"host" env:"DB_HOST"`
	Port             int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User             string `yaml:"user" env:"DB_USER"`
	Password         string `yaml:"password" env:"DB_PASSWORD"`
	Name             string `yaml:"name" env:"DB_NAME"`
	SSLMode          string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-default:"disable"`
	ConnectionString string `yaml:"connection_string" env:"DB_CONNECTION_STRING"`
}

// LoadConfig reads configuration from a YAML file and overrides with Env variables
func LoadConfig(configPath string) (*Config, error) {
	var cfg Config

	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	// This fills in any missing pieces from Env variables and applies defaults
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("error reading env variables: %w", err)
	}

	return &cfg, nil
}
