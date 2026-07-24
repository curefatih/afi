package gatewayconfig

import (
	"fmt"
	"strings"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// ObjectStoreConfig is per-org optional S3-compatible asset persistence.
// Nil or Enabled=false means the gateway does not persist generated assets.
type ObjectStoreConfig struct {
	Enabled           bool   `json:"enabled"`
	Endpoint          string `json:"endpoint,omitempty"`
	Region            string `json:"region,omitempty"`
	Bucket            string `json:"bucket,omitempty"`
	UseSSL            bool   `json:"use_ssl"`
	PathStyle         bool   `json:"path_style"`
	CredentialID      string `json:"credential_id,omitempty"`
	AccessKeyEnv      string `json:"access_key_env,omitempty"`
	SecretKeyEnv      string `json:"secret_key_env,omitempty"`
	PresignTTLSeconds int    `json:"presign_ttl_seconds,omitempty"`
}

// NormalizeObjectStore validates and returns a canonical config.
// A nil input returns (nil, nil). Clearing persistence uses enabled:false (not nil) or DELETE semantics via nil.
func NormalizeObjectStore(c *ObjectStoreConfig) (*ObjectStoreConfig, error) {
	if c == nil {
		return nil, nil
	}
	out := *c
	out.Endpoint = strings.TrimSpace(out.Endpoint)
	out.Region = strings.TrimSpace(out.Region)
	out.Bucket = strings.TrimSpace(out.Bucket)
	out.CredentialID = strings.TrimSpace(out.CredentialID)
	out.AccessKeyEnv = strings.TrimSpace(out.AccessKeyEnv)
	out.SecretKeyEnv = strings.TrimSpace(out.SecretKeyEnv)
	if out.PresignTTLSeconds < 0 {
		return nil, fmt.Errorf("%w: presign_ttl_seconds must be >= 0", kernel.ErrInvalidRequest)
	}
	if !out.Enabled {
		return &out, nil
	}
	if out.Endpoint == "" {
		return nil, fmt.Errorf("%w: endpoint is required when object store is enabled", kernel.ErrInvalidRequest)
	}
	if out.Bucket == "" {
		return nil, fmt.Errorf("%w: bucket is required when object store is enabled", kernel.ErrInvalidRequest)
	}
	if out.CredentialID == "" && (out.AccessKeyEnv == "" || out.SecretKeyEnv == "") {
		return nil, fmt.Errorf("%w: credential_id or both access_key_env and secret_key_env are required when enabled", kernel.ErrInvalidRequest)
	}
	if out.PresignTTLSeconds == 0 {
		out.PresignTTLSeconds = 3600
	}
	if out.Region == "" {
		out.Region = "us-east-1"
	}
	return &out, nil
}

// ToSnapshot copies this config into the compiled snapshot shape.
func (c *ObjectStoreConfig) ToSnapshot() *snapshot.ObjectStoreConfig {
	if c == nil {
		return nil
	}
	return &snapshot.ObjectStoreConfig{
		Enabled:           c.Enabled,
		Endpoint:          c.Endpoint,
		Region:            c.Region,
		Bucket:            c.Bucket,
		UseSSL:            c.UseSSL,
		PathStyle:         c.PathStyle,
		CredentialID:      c.CredentialID,
		AccessKeyEnv:      c.AccessKeyEnv,
		SecretKeyEnv:      c.SecretKeyEnv,
		PresignTTLSeconds: c.PresignTTLSeconds,
	}
}
