package wasm

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config configures an S3-compatible object store for module_uri s3://bucket/key.
type S3Config struct {
	Endpoint  string // host:port (no scheme), e.g. localhost:9000
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
	// PathStyle forces path-style addressing (typical for MinIO).
	PathStyle bool
}

// BlobFetcher loads module bytes from remote URIs (s3://…).
type BlobFetcher interface {
	Fetch(ctx context.Context, moduleURI string) ([]byte, error)
}

// S3Fetcher fetches objects from an S3-compatible endpoint.
type S3Fetcher struct {
	client *minio.Client
}

// NewS3Fetcher builds a fetcher. Returns nil, nil when cfg.Endpoint is empty.
func NewS3Fetcher(cfg S3Config) (*S3Fetcher, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, nil
	}
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: region,
	}
	if cfg.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("wasm s3: %w", err)
	}
	return &S3Fetcher{client: client}, nil
}

// Fetch downloads s3://bucket/key.
func (f *S3Fetcher) Fetch(ctx context.Context, moduleURI string) ([]byte, error) {
	if f == nil || f.client == nil {
		return nil, fmt.Errorf("wasm: s3 fetcher not configured")
	}
	bucket, key, err := parseS3URI(moduleURI)
	if err != nil {
		return nil, err
	}
	obj, err := f.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("wasm s3 get %s: %w", moduleURI, err)
	}
	defer obj.Close()
	b, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("wasm s3 read %s: %w", moduleURI, err)
	}
	return b, nil
}

func parseS3URI(moduleURI string) (bucket, key string, err error) {
	u, err := url.Parse(strings.TrimSpace(moduleURI))
	if err != nil {
		return "", "", fmt.Errorf("wasm: s3 uri: %w", err)
	}
	if u.Scheme != "s3" {
		return "", "", fmt.Errorf("wasm: expected s3:// uri, got %q", u.Scheme)
	}
	bucket = u.Host
	key = strings.TrimPrefix(u.Path, "/")
	if bucket == "" || key == "" {
		return "", "", fmt.Errorf("wasm: s3 uri must be s3://bucket/key")
	}
	return bucket, key, nil
}
