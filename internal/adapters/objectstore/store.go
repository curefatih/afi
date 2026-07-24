// Package objectstore provides an S3-compatible object store for generated assets.
package objectstore

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config configures an S3-compatible endpoint.
type Config struct {
	Endpoint  string // host:port (no scheme), e.g. localhost:9000
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string
	UseSSL    bool
	PathStyle bool
}

// PutOptions controls object metadata on write.
type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

// Store is the asset persistence port.
type Store interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, opts PutOptions) error
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// S3 is a MinIO/S3-backed Store.
type S3 struct {
	client *minio.Client
	bucket string
}

// New builds an S3 store. Returns nil, nil when Endpoint or Bucket is empty.
func New(cfg Config) (*S3, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	bucket := strings.TrimSpace(cfg.Bucket)
	if endpoint == "" || bucket == "" {
		return nil, nil
	}
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	region := strings.TrimSpace(cfg.Region)
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
		return nil, fmt.Errorf("objectstore s3: %w", err)
	}
	return &S3{client: client, bucket: bucket}, nil
}

// Put uploads an object.
func (s *S3) Put(ctx context.Context, key string, body io.Reader, size int64, opts PutOptions) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("objectstore: not configured")
	}
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return fmt.Errorf("objectstore: empty key")
	}
	putOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
	}
	if putOpts.ContentType == "" {
		putOpts.ContentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, putOpts)
	if err != nil {
		return fmt.Errorf("objectstore put %s: %w", key, err)
	}
	return nil
}

// PresignGet returns a time-limited GET URL.
func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("objectstore: not configured")
	}
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return "", fmt.Errorf("objectstore: empty key")
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("objectstore presign %s: %w", key, err)
	}
	return u.String(), nil
}

// AssetKey builds an org/project-scoped object key.
func AssetKey(orgID, projectID, assetID, ext string) string {
	orgID = strings.TrimSpace(orgID)
	projectID = strings.TrimSpace(projectID)
	assetID = strings.TrimSpace(assetID)
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if projectID == "" {
		projectID = "_org"
	}
	if ext == "" {
		ext = "bin"
	}
	return fmt.Sprintf("%s/%s/%s.%s", orgID, projectID, assetID, ext)
}
