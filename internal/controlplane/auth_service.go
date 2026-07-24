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
	afisaml "github.com/curefatih/afi/internal/adapters/saml"
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
func NewAuthService(
	cfg *kernel.Config,
	users identity.UserRepository,
	identities identity.ExternalIdentityRepository,
	resets identity.PasswordResetRepository,
	states identity.SSOStateStore,
) *platform.AuthService {
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
		Resets:        resets,
		Tokens:        authAdapter,
		Passwords:     authAdapter,
		States:        states,
		Providers:     map[string]identity.FederationProvider{},
		SSOEnabled:    cfg.Auth.SSO.Enabled,
		SignupEnabled: cfg.Auth.SignupEnabled,
		PublicBaseURL: cfg.Auth.PublicBaseURL,
		AppBaseURL:    cfg.Mail.PublicAppURL,
		NewUserID:     func() string { return newID("user") },
		NewLinkID:     func() string { return newID("extid") },
		NewResetID:    func() string { return newID("pwreset") },
	}
	if !cfg.Auth.SSO.Enabled {
		return svc
	}
	log := slog.Default()
	for _, p := range cfg.Auth.SSO.Providers {
		requireVerified := true
		if p.RequireEmailVerified != nil {
			requireVerified = *p.RequireEmailVerified
		}
		typ := strings.ToLower(strings.TrimSpace(p.Type))
		var (
			prov identity.FederationProvider
			err  error
		)
		switch typ {
		case "saml":
			metaURL := strings.TrimRight(cfg.Auth.PublicBaseURL, "/") + "/api/v1/platform/auth/sso/" + p.ID + "/metadata"
			acsURL := strings.TrimRight(cfg.Auth.PublicBaseURL, "/") + "/api/v1/platform/auth/sso/" + p.ID + "/callback"
			var sp *afisaml.Provider
			sp, err = afisaml.New(afisaml.Config{
				ID:                   p.ID,
				DisplayName:          p.DisplayName,
				EntityID:             p.EntityID,
				MetadataURL:          metaURL,
				ACSURL:               acsURL,
				IDPMetadataURL:       p.IDPMetadataURL,
				IDPMetadataXML:       p.IDPMetadataXML,
				SPCertPEM:            p.SPCertPEM,
				SPKeyPEM:             p.SPKeyPEM,
				RequireEmailVerified: requireVerified,
				AllowIDPInitiated:    true,
			})
			if err == nil {
				prov = sp
				if sp.EphemeralKey() && log != nil {
					log.Warn("sso saml sp key is ephemeral; set sp_cert_pem/sp_key_pem for stable metadata", "id", p.ID)
				}
			}
		default:
			prov, err = oauthoidc.New(oauthoidc.Config{
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
		}
		if err != nil {
			if log != nil {
				log.Error("sso provider skipped", "id", p.ID, "type", typ, "err", err)
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
