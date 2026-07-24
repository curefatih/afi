package gatewayconfig

import "testing"

func TestNormalizeObjectStoreNil(t *testing.T) {
	got, err := NormalizeObjectStore(nil)
	if err != nil || got != nil {
		t.Fatalf("got %#v err=%v", got, err)
	}
}

func TestNormalizeObjectStoreDisabled(t *testing.T) {
	got, err := NormalizeObjectStore(&ObjectStoreConfig{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled {
		t.Fatal("expected disabled")
	}
}

func TestNormalizeObjectStoreEnabledRequiresEndpoint(t *testing.T) {
	_, err := NormalizeObjectStore(&ObjectStoreConfig{
		Enabled: true, Bucket: "b", AccessKeyEnv: "A", SecretKeyEnv: "S",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeObjectStoreEnabledOK(t *testing.T) {
	got, err := NormalizeObjectStore(&ObjectStoreConfig{
		Enabled: true, Endpoint: "localhost:9000", Bucket: "afi",
		AccessKeyEnv: "AK", SecretKeyEnv: "SK",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.PresignTTLSeconds != 3600 || got.Region != "us-east-1" {
		t.Fatalf("defaults: %+v", got)
	}
}
