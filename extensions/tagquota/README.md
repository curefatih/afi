# tagquota (example)

Example `BeforeCall` hook that enforces a request limit per `X-AFI-Tags` value (e.g. each `end-user-id` gets its own counter).

**Not shipped as a gateway built-in.** The control plane Quotas UI and snapshot quotas are unchanged. Use this package as a reference when writing your own hook or external policy service.

## Wire it yourself

In `cmd/gateway` (or your own gateway binary), after counters are configured:

```go
import "github.com/curefatih/afi/extensions/tagquota"

hooks.RegisterBeforeCall(tagquota.New(pipeline.Counters, tagquota.Config{
    TagKey: "end-user-id",
    Limit:  100,
    Window: "total", // or minute / hour / day (Redis)
}))
```

Requests without that tag key are not limited by this hook.

See also [`docs/hooks/usage.md`](../../docs/hooks/usage.md) and [`sdk/hook`](../../sdk/hook).
