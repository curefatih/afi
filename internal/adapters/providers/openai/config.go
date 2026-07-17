package openai

import "net/url"

type Config struct {
	BaseURL *url.URL

	APIKey string
}
