package secrets_test

import (
	"testing"

	"github.com/curefatih/afi/internal/adapters/secrets"
)

func TestParseRef(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in     string
		scheme string
		path   string
		key    string
	}{
		{"OPENAI_API_KEY", "env", "OPENAI_API_KEY", ""},
		{"env://OPENAI_API_KEY", "env", "OPENAI_API_KEY", ""},
		{"aws-sm://us-east-1/afi/openai#api_key", "aws-sm", "us-east-1/afi/openai", "api_key"},
		{"hashicorp://secret/data/afi/openai#value", "hashicorp", "secret/data/afi/openai", "value"},
	}
	for _, tc := range cases {
		p, err := secrets.ParseRef(tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if p.Scheme != tc.scheme || p.Path != tc.path || p.Key != tc.key {
			t.Fatalf("%s: got scheme=%q path=%q key=%q", tc.in, p.Scheme, p.Path, p.Key)
		}
	}
}
