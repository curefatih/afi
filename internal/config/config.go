package config

import "time"

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
	TimeoutMS int `yaml:"timeout_ms"`
}

type Config struct {
	Server    ServerConfig        `yaml:"server"`
	Providers map[string]Provider `yaml:"providers"`
	Hooks     HooksConfig         `yaml:"hooks"`
}
