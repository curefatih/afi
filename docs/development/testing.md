# Testing

## Unit tests (test-first)

```bash
make test
# or
go test ./...
```

Covered areas:

* Snapshot compile + org-scoped `LookupRoute` + hashed `LookupKey` + quota resolve
* Data-plane auth, quota 429 / under-limit, mock OpenAI/Anthropic chat, route failover
* Usage outbox worker `ProcessOnce` + cost calculation

* Control-plane JWT/password helpers
* Internal token checks
* Org membership middleware (403 when not a member)
* Publish error surfaced on create project
* Migrate wipe policy (version bumps never wipe)

## Automated local verify

With Postgres + control plane + gateway already running:

```bash
make verify
```

Script: [`scripts/verify-local.sh`](../../scripts/verify-local.sh).

Checks health, login, bad key → 401, internal publish auth, snapshot hot reload, optional live chat when `OPENAI_API_KEY` is set, optional usage drain when the worker is running, and quota → 429.

## Manual smoke

See [Verify](../getting-started/verify.md) for curl recipes including UI-edited routes.

## Web

```bash
pnpm --dir web test
pnpm --dir web check
```
