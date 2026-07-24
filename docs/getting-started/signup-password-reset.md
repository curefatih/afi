# Self-serve signup and password reset

Platform users can create an account with email/password when an operator enables signup, and can recover a forgotten password via email.

!!! note
    These flows authenticate **platform UI / control-plane** users only. Gateway inference still uses **virtual API keys**.

## Sign up (end users)

1. Open the platform UI (local default: http://localhost:3000).
2. On the login page, use **Create account** (only shown when signup is enabled).
3. Enter name, email, and password (at least 8 characters).
4. You are signed in and landed in the app. Create or join an organization as usual — signup does **not** create an org automatically.

If signup is disabled, the signup page explains that an administrator must provision the account (invite or seed).

## Password reset (end users)

1. On the login page, click **Forgot password?**
2. Enter your email. AFI always shows a success message (it does not reveal whether the address exists).
3. Open the link in the email (`/auth/reset-password/{token}`). Links expire after **one hour**.
4. Choose a new password. You are signed in with a new session.

SSO-only accounts (no local password) can also set a password through this flow.

## Operator setup

### 1. Enable signup (optional)

```yaml
auth:
  signup_enabled: true   # default false
```

| Env | YAML | Notes |
|-----|------|--------|
| `AFI_SIGNUP_ENABLED` | `auth.signup_enabled` | Turn self-serve registration on/off |

Password reset has **no** separate flag. It uses the deployment mail sender (same stack as org invites).

### 2. Mail

Configure outbound mail so reset emails leave the control plane:

```yaml
mail:
  public_app_url: "https://afi.example.com"   # web UI origin for reset links
  from: "AFI <noreply@example.com>"
  default_provider: smtp                      # or resend | log
  smtp:
    enabled: true
    host: smtp.example.com
    port: 587
    # ...
```

| Env | YAML | Notes |
|-----|------|--------|
| `AFI_MAIL_PUBLIC_APP_URL` | `mail.public_app_url` | Base URL embedded in reset links |
| `AFI_MAIL_DEFAULT_PROVIDER` | `mail.default_provider` | `smtp`, `resend`, or `log` (dev) |
| `AFI_MAIL_FROM` | `mail.from` | From header |

With `default_provider: log` (or no SMTP/Resend), reset mail is written to control-plane logs — useful locally.

Restart the **control plane** after changing auth or mail config.

### 3. Public APIs

| Method | Path | Auth |
|--------|------|------|
| `GET` | `/api/v1/platform/auth/features` | none |
| `POST` | `/api/v1/platform/auth/register` | none |
| `POST` | `/api/v1/platform/auth/password-reset` | none |
| `POST` | `/api/v1/platform/auth/password-reset/{token}` | none |

See also [SSO](sso.md) for federated login.
