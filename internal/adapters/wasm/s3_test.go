package wasm

import "testing"

func TestParseS3URI(t *testing.T) {
	bucket, key, err := parseS3URI("s3://afi-hooks/org1/block.wasm")
	if err != nil {
		t.Fatal(err)
	}
	if bucket != "afi-hooks" || key != "org1/block.wasm" {
		t.Fatalf("got %q %q", bucket, key)
	}
	if _, _, err := parseS3URI("https://example.com/x"); err == nil {
		t.Fatal("expected scheme error")
	}
	if _, _, err := parseS3URI("s3://bucketonly"); err == nil {
		t.Fatal("expected key error")
	}
}

func TestNewS3FetcherEmpty(t *testing.T) {
	f, err := NewS3Fetcher(S3Config{})
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Fatal("expected nil fetcher when endpoint empty")
	}
}

func TestModuleCacheS3RequiresFetcher(t *testing.T) {
	cache := NewModuleCache(Config{Name: "s3-test"})
	defer func() { _ = cache.Close(t.Context()) }()
	_, err := cache.Get(t.Context(), "s3://b/k.wasm", "", "x")
	if err == nil {
		t.Fatal("expected error without fetcher")
	}
}
