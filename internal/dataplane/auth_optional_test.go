package dataplane

import (
	"context"
	"log/slog"
	"testing"

	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestBindProviderSecretAuthOptional(t *testing.T) {
	t.Parallel()
	p := &Pipeline{Log: slog.Default()}
	prov := snapshot.Provider{ID: "b", Type: "bedrock", APIKeyEnv: ""}
	got, credID, err := p.bindProviderSecret(context.Background(), &snapshot.Snapshot{}, prov, snapshot.APIKey{}, policy.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if credID != "" {
		t.Fatalf("credID=%q", credID)
	}
	if got.Type != "bedrock" {
		t.Fatalf("got=%+v", got)
	}

	openai := snapshot.Provider{ID: "o", Type: "openai", APIKeyEnv: ""}
	_, _, err = p.bindProviderSecret(context.Background(), &snapshot.Snapshot{}, openai, snapshot.APIKey{}, policy.Request{})
	if err == nil {
		t.Fatal("expected error for openai without api_key_env")
	}
}
