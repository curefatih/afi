package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	goredis "github.com/redis/go-redis/v9"
)

const ssoStateKeyPrefix = "afi:sso:state:"

// SSOStateStore stores SSO CSRF state in Redis for multi-node control planes.
type SSOStateStore struct {
	Client *goredis.Client
	TTL    time.Duration
	Now    func() time.Time
}

func NewSSOStateStore(client *goredis.Client, ttl time.Duration) *SSOStateStore {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &SSOStateStore{Client: client, TTL: ttl}
}

func (s *SSOStateStore) Put(ctx context.Context, state string, value identity.SSOState) error {
	if s == nil || s.Client == nil {
		return fmt.Errorf("redis sso state store not configured")
	}
	if state == "" {
		return kernel.ErrInvalidRequest
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	if value.ExpiresAt.IsZero() {
		value.ExpiresAt = now.Add(s.TTL)
	}
	ttl := value.ExpiresAt.Sub(now)
	if ttl <= 0 {
		ttl = s.TTL
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.Client.Set(ctx, ssoStateKeyPrefix+state, payload, ttl).Err()
}

func (s *SSOStateStore) Take(ctx context.Context, state string) (identity.SSOState, error) {
	if s == nil || s.Client == nil {
		return identity.SSOState{}, fmt.Errorf("redis sso state store not configured")
	}
	if state == "" {
		return identity.SSOState{}, kernel.ErrNotFound
	}
	key := ssoStateKeyPrefix + state
	// GETDEL is atomic (Redis >= 6.2) so begin/callback across replicas stays one-shot.
	raw, err := s.Client.GetDel(ctx, key).Bytes()
	if err == goredis.Nil {
		return identity.SSOState{}, kernel.ErrNotFound
	}
	if err != nil {
		return identity.SSOState{}, err
	}
	var value identity.SSOState
	if err := json.Unmarshal(raw, &value); err != nil {
		return identity.SSOState{}, err
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	if !value.ExpiresAt.IsZero() && now.After(value.ExpiresAt) {
		return identity.SSOState{}, kernel.ErrNotFound
	}
	return value, nil
}

var _ identity.SSOStateStore = (*SSOStateStore)(nil)
