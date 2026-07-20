# Single sign-on (SSO)

Platform users can sign in to the AFI control plane with an external identity provider (IdP) using **OAuth 2.0** or **OpenID Connect (OIDC)**. After a successful IdP login, AFI issues the same session JWT used by email/password login.

!!! note
    SSO authenticates **platform UI / control-plane** users only. Gateway inference still uses **virtual API keys**. SAML 2.0 is reserved for a later release (same federation ports).

## Sign in (end users)

1. Open the platform UI (local default: http://localhost:3000).
2. On the login page, use **Sign in** with email/password, or click a configured IdP button (for example **Google**).
3. Complete login at the IdP.
4. You are redirected back to AFI and landed in the app.

If your email already has an AFI account, SSO links that account. Otherwise AFI **JIT-provisions** a new member user (no local password). New users are not added to an organization automatically — an admin must invite them, or they create/join an org as usual.

## Operator setup

### 1. URLs the IdP must know

| Purpose | URL |
|---------|-----|
| OAuth redirect (callback) | `{auth.public_base_url}/api/v1/platform/auth/sso/{provider_id}/callback` |
| Web app after login | `{mail.public_app_url}/auth/sso/callback` |

Examples (local):

* Callback: `http://localhost:8081/api/v1/platform/auth/sso/google/callback`
* App return: `http://localhost:3000/auth/sso/callback`

Register the **callback** URL in your IdP as an allowed redirect URI. `auth.public_base_url` must be the URL clients and the IdP can reach for the control plane (not an internal-only listen address behind NAT without a reverse proxy).

### 2. Config

In your AFI config (for example `configs/local.yaml` / production YAML):

```yaml
auth:
  jwt_secret: "..."          # change in production
  token_ttl: "24h"
  public_base_url: "https://afi-control.example.com"
  sso:
    enabled: true
    # redis = shared CSRF state across control-plane replicas (recommended)
    # memory = single node / local tests only
    state_store: redis
    providers:
      - id: google
        type: oidc                 # oidc | oauth2
        display_name: Google
        issuer: https://accounts.google.com
        client_id: "YOUR_CLIENT_ID"
        client_secret: "YOUR_CLIENT_SECRET"
        scopes: ["openid", "email", "profile"]
        require_email_verified: true
```

Environment overrides:

| Env | YAML | Notes |
|-----|------|--------|
| `AFI_SSO_ENABLED` | `auth.sso.enabled` | Turn SSO on/off |
| `AFI_SSO_STATE_STORE` | `auth.sso.state_store` | `redis` (default) or `memory` |
| `AFI_AUTH_PUBLIC_BASE_URL` | `auth.public_base_url` | Control-plane public URL for callbacks |
| `AFI_REDIS_URL` | `redis_url` | Required when `state_store=redis` |
| `AFI_MAIL_PUBLIC_APP_URL` | `mail.public_app_url` | Web UI origin for post-login redirect |

Provider `client_id` / `client_secret` are YAML-only today (put secrets in a private config or secret-mounted file; do not commit them).

Restart the **control plane** after changing SSO config.

### 3. Horizontal scale

With multiple control-plane replicas, keep `auth.sso.state_store: redis` so the OAuth `state` cookie-equivalent is shared. `memory` only works if begin and callback hit the same process.

Redis is already used by the gateway for timed quotas; the same `redis_url` serves SSO CSRF keys under `afi:sso:state:*` (TTL ~10 minutes).

## Provider types

### OIDC (`type: oidc`)

Set `issuer` to the IdP issuer URL. AFI discovers authorization, token, and userinfo endpoints from `/.well-known/openid-configuration`. Default scopes: `openid`, `email`, `profile`. Email verification is required by default (`require_email_verified: true`).

Works with common IdPs that speak OIDC (Google, Okta, Azure AD / Entra ID, Auth0, Keycloak, …).

### OAuth 2.0 (`type: oauth2`)

For providers without OIDC discovery, set endpoints explicitly:

```yaml
- id: myoauth
  type: oauth2
  display_name: My OAuth
  client_id: "..."
  client_secret: "..."
  auth_url: https://idp.example/oauth/authorize
  token_url: https://idp.example/oauth/token
  userinfo_url: https://idp.example/oauth/userinfo
  scopes: ["email", "profile"]
  require_email_verified: false
```

When `require_email_verified` is `false`, a present email from a successful token exchange is treated as acceptable for JIT.

### SAML 2.0

Not implemented yet. The identity ports leave room for a future SAML adapter; do not set `type: saml` today.

## How provisioning works (JIT)

On first successful SSO for a given IdP subject:

1. If `(provider, subject)` is already linked → sign in as that user.
2. Else if a **verified** email matches an existing user → link the IdP identity and sign in.
3. Else create a new user (`role: member`, no password) and link the identity.
4. Unverified email with no prior link → login rejected.

Password login still works for users that have a password hash. Federated-only users cannot use email/password until a password is set through another flow.

## API surface

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/platform/auth/sso/providers` | List enabled providers for the login UI |
| `GET` | `/api/v1/platform/auth/sso/{provider}/start` | Begin SSO (redirects to IdP) |
| `GET` | `/api/v1/platform/auth/sso/{provider}/callback` | IdP callback → redirect to web with `token` or `error` |
| `POST` | `/api/v1/platform/auth/login` | Email/password (unchanged) |
| `GET` | `/api/v1/platform/auth/me` | Current user (Bearer JWT) |

Optional query on start: `?redirect=/app/dashboard` (relative path only) — restored after callback.

## Checklist

1. Set strong `auth.jwt_secret` and production `public_base_url` / `mail.public_app_url`.
2. Create an OAuth/OIDC app in the IdP; add the AFI callback URL.
3. Enable `auth.sso` with at least one provider and restart control plane.
4. Confirm Redis is reachable if `state_store=redis`.
5. Open the web login page and confirm the IdP button appears.
6. Complete a test login; confirm `/api/v1/platform/auth/me` works with the issued token.

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| No SSO buttons on login | `sso.enabled=false`, no valid providers, or control plane not restarted |
| `redirect_uri_mismatch` at IdP | Callback URL does not exactly match `public_base_url` + `/api/v1/platform/auth/sso/{id}/callback` |
| `invalid or expired sso state` | State TTL expired (~10m), `state_store=memory` with multiple replicas, or Redis unreachable |
| `email not verified` | IdP did not assert verified email; set `require_email_verified: false` only if you accept that risk |
| User signed in but sees no orgs | Expected for JIT — invite the user or have them create an organization |

Related: [Web UI](web-ui.md), [Customization reference](../deployment/customization.md), [Config reference](../development/config-reference.md).
