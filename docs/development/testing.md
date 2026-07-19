# Testing

## Run all Go tests

```bash
make test
# or
go test ./...
```

## What is covered

* Snapshot compile helpers
* Data-plane auth (reject unknown keys)
* OpenAI adapter non-stream path with a fake HTTP upstream
* Control-plane password hash / JWT helpers where present

## Manual smoke

Follow [Verify](../getting-started/verify.md) against a running stack.

## Web

```bash
pnpm --dir web test
pnpm --dir web check
```
