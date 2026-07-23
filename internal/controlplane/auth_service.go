package controlplane

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	adapterauth "github.com/curefatih/afi/internal/adapters/auth"
	"github.com/curefatih/afi/internal/adapters/memory"
	"github.com/curefatih/afi/internal/adapters/oauthoidc"
	afiredis "github.com/curefatih/afi/internal/adapters/redis"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	goredis "github.com/redis/go-redis/v9"
)

const defaultSSOStateTTL = 10 * time.Minute

// NewSSOStateStore builds the CSRF state backend for SSO.
// Prefer redis for horizontally scaled control planes; memory is for local/tests only.
func NewSSOStateStore(cfg *kernel.Config, rdb *goredis.Client) (identity.SSOStateStore, error) {
	backend := "redis"
	if cfg != nil && cfg.Auth.SSO.StateStore != "" {
		backend = strings.ToLower(strings.TrimSpace(cfg.Auth.SSO.StateStore))
	}
	switch backend {
	case "memory":
		return memory.NewSSOStateStore(defaultSSOStateTTL), nil
	case "redis":
		if rdb == nil {
			return nil, fmt.Errorf("auth.sso.state_store=redis requires a redis client (check redis_url)")
		}
		return afiredis.NewSSOStateStore(rdb, defaultSSOStateTTL), nil
	default:
		return nil, fmt.Errorf("unknown auth.sso.state_store %q", backend)
	}
}

// NewAuthService wires password + SSO auth against identity ports (composition root supplies adapters).
func NewAuthService(cfg *kernel.Config, users identity.UserRepository, identities identity.ExternalIdentityRepository, states identity.SSOStateStore) *platform.AuthService {
	if cfg == nil || users == nil {
		return nil
	}
	if states == nil && cfg.Auth.SSO.Enabled {
		// Fallback for tests / miswired composition roots when SSO is on.
		states = memory.NewSSOStateStore(defaultSSOStateTTL)
	}
	authAdapter := adapterauth.NewService(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	svc := &platform.AuthService{
		Users:         users,
		Identities:    identities,
		Tokens:        authAdapter,
		Passwords:     authAdapter,
		States:        states,
		Providers:     map[string]identity.FederationProvider{},
		SSOEnabled:    cfg.Auth.SSO.Enabled,
		PublicBaseURL: cfg.Auth.PublicBaseURL,
		AppBaseURL:    cfg.Mail.PublicAppURL,
		NewUserID:     func() string { return newID("user") },
		NewLinkID:     func() string { return newID("extid") },
	}
	if !cfg.Auth.SSO.Enabled {
		return svc
	}
	for _, p := range cfg.Auth.SSO.Providers {
		requireVerified := true
		if p.RequireEmailVerified != nil {
			requireVerified = *p.RequireEmailVerified
		}
		prov, err := oauthoidc.New(oauthoidc.Config{
			ID:                   p.ID,
			Type:                 p.Type,
			DisplayName:          p.DisplayName,
			Issuer:               p.Issuer,
			ClientID:             p.ClientID,
			ClientSecret:         p.ClientSecret,
			Scopes:               p.Scopes,
			AuthURL:              p.AuthURL,
			TokenURL:             p.TokenURL,
			UserInfoURL:          p.UserInfoURL,
			RequireEmailVerified: requireVerified,
		})
		if err != nil {
			if log := slog.Default(); log != nil {
				log.Error("sso provider skipped", "id", p.ID, "err", err)
			}
			continue
		}
		svc.Providers[prov.Name()] = prov
	}
	if len(svc.Providers) == 0 {
		svc.SSOEnabled = false
	}
	return svc
}

// EnsureSSOConfigured returns an error when SSO is enabled but no providers loaded.
func EnsureSSOConfigured(cfg *kernel.Config, auth *platform.AuthService) error {
	if cfg == nil || !cfg.Auth.SSO.Enabled {
		return nil
	}
	if auth == nil || len(auth.Providers) == 0 {
		return fmt.Errorf("auth.sso.enabled but no valid providers configured")
	}
	return nil
}
