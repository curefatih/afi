package credentials

import (
	"encoding/base64"
	"testing"
)

func TestBoxRoundTrip(t *testing.T) {
	t.Parallel()
	box, err := ParseMasterKey("local-dev-master-key")
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("sk-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	got, err := box.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-test-secret" {
		t.Fatalf("got %q", got)
	}
}

func TestParseMasterKeyBase64(t *testing.T) {
	t.Parallel()
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	box, err := ParseMasterKey("base64:" + base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("x")
	if err != nil {
		t.Fatal(err)
	}
	got, err := box.Open(sealed)
	if err != nil || got != "x" {
		t.Fatalf("got %q err=%v", got, err)
	}
}
