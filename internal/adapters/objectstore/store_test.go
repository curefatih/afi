package objectstore

import "testing"

func TestAssetKey(t *testing.T) {
	got := AssetKey("org1", "proj1", "asset1", "png")
	if got != "org1/proj1/asset1.png" {
		t.Fatalf("got %q", got)
	}
	got = AssetKey("org1", "", "a", ".jpg")
	if got != "org1/_org/a.jpg" {
		t.Fatalf("got %q", got)
	}
}

func TestNewEmpty(t *testing.T) {
	s, err := New(Config{})
	if err != nil || s != nil {
		t.Fatalf("expected nil store, got %#v err=%v", s, err)
	}
	s, err = New(Config{Endpoint: "localhost:9000"})
	if err != nil || s != nil {
		t.Fatalf("expected nil without bucket, got %#v err=%v", s, err)
	}
}
