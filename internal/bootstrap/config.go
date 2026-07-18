package bootstrap

import (
	"net/url"
	"os"
)

type Config struct {
	HTTP HTTPConfig

	OpenAI OpenAIConfig

	Database DatabaseConfig

	Redis RedisConfig
}

type HTTPConfig struct {
	Address string
}

type OpenAIConfig struct {
	BaseURL *url.URL

	APIKey string
}

type DatabaseConfig struct {
	DSN string
}

type RedisConfig struct {
	Address string

	Password string

	DB int
}

func loadConfig() (Config, error) {
	openAIBaseURL, err := url.Parse(
		env("OPENAI_BASE_URL", "https://api.openai.com"),
	)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP: HTTPConfig{
			Address: env("HTTP_ADDRESS", ":8080"),
		},

		OpenAI: OpenAIConfig{
			BaseURL: openAIBaseURL,
			APIKey:  os.Getenv("OPENAI_API_KEY"),
		},

		Database: DatabaseConfig{
			DSN: os.Getenv("DATABASE_DSN"),
		},

		Redis: RedisConfig{
			Address:  env("REDIS_ADDRESS", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       envInt("REDIS_DB", 0),
		},
	}, nil
}
