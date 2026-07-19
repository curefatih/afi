# Testing

## Unit tests (test-first)

```bash
make test
# or
go test ./...
```

Covered areas:

* Snapshot compile + org-scoped `LookupRoute` + hashed `LookupKey`
* Data-plane auth (reject unknown keys) and mock upstream chat
* Usage token parsing on non-stream success
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

Checks health, login, bad key → 401, internal publish auth, snapshot hot reload, optional live chat when `OPENAI_API_KEY` is set.

## Manual smoke

See [Verify](../getting-started/verify.md) for curl recipes including UI-edited routes.

## Web

```bash
pnpm --dir web test
pnpm --dir web check
```
