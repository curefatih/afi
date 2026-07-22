package secrets_test

import (
	"context"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestEnvGet(t *testing.T) {
	t.Setenv("AFI_TEST_SECRET", "value-1")
	env := secrets.Env{}
	got, err := env.Get(context.Background(), "AFI_TEST_SECRET")
	if err != nil || got != "value-1" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	if _, err := env.Get(context.Background(), ""); err == nil {
		t.Fatal("expected empty ref error")
	}
	if _, err := env.Get(context.Background(), "AFI_MISSING_SECRET_XYZ"); err == nil {
		t.Fatal("expected missing env error")
	}
}

func TestCredentialResolverEnvAndEncrypted(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-env")
	box, err := credentials.ParseMasterKey("test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("sk-sealed")
	if err != nil {
		t.Fatal(err)
	}
	r := secrets.NewCredentialResolver(box)

	envSecret, err := r.Open(context.Background(), snapshot.Credential{
		StorageKind: snapshot.CredentialStorageEnv,
		SecretRef:   "OPENAI_API_KEY",
	})
	if err != nil || envSecret != "sk-env" {
		t.Fatalf("env=%q err=%v", envSecret, err)
	}

	dbSecret, err := r.Open(context.Background(), snapshot.Credential{
		StorageKind:      snapshot.CredentialStorageEncryptedDB,
		EncryptedPayload: sealed,
	})
	if err != nil || dbSecret != "sk-sealed" {
		t.Fatalf("db=%q err=%v", dbSecret, err)
	}

	_, err = r.Open(context.Background(), snapshot.Credential{StorageKind: "vault", SecretRef: "aws-sm://us-east-1/x"})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		// Multi with only Env: aws-sm scheme should fail as not configured
		t.Fatalf("err=%v", err)
	}

	fake := secrets.Multi{
		Env: secrets.Env{},
		AWSSM: resolverFunc(func(_ context.Context, ref string) (string, error) {
			return "sk-from-vault", nil
		}),
	}
	r = secrets.NewCredentialResolver(box).WithVault(fake)
	vaultSecret, err := r.Open(context.Background(), snapshot.Credential{
		StorageKind: snapshot.CredentialStorageVault,
		SecretRef:   "aws-sm://us-east-1/afi/openai#api_key",
	})
	if err != nil || vaultSecret != "sk-from-vault" {
		t.Fatalf("vault=%q err=%v", vaultSecret, err)
	}

	noBox := secrets.NewCredentialResolver(nil)
	_, err = noBox.Open(context.Background(), snapshot.Credential{
		StorageKind:      snapshot.CredentialStorageEncryptedDB,
		EncryptedPayload: sealed,
	})
	if err == nil || !strings.Contains(err.Error(), "master key") {
		t.Fatalf("err=%v", err)
	}
}

type resolverFunc func(context.Context, string) (string, error)

func (f resolverFunc) Get(ctx context.Context, ref string) (string, error) { return f(ctx, ref) }
