package controlplane

import (
	"fmt"
	"log/slog"

	adapterauth "github.com/curefatih/afi/internal/adapters/auth"
	"github.com/curefatih/afi/internal/adapters/memory"
	"github.com/curefatih/afi/internal/adapters/oauthoidc"
	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

func newAuthService(cfg *kernel.Config, store *Store) *platform.AuthService {
	if cfg == nil || store == nil {
		return nil
	}
	authAdapter := adapterauth.NewService(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	svc := &platform.AuthService{
		Users:         postgres.NewUsers(store.pool),
		Identities:    postgres.NewExternalIdentities(store.pool),
		Tokens:        authAdapter,
		Passwords:     authAdapter,
		States:        memory.NewSSOStateStore(0),
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
