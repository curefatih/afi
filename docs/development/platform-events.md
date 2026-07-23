# Platform domain events

Control-plane mutations emit in-process `platform.Event` values on `platform.Bus`. Optionally, the same events are durably enqueued and published cross-process.

## Flow

```text
HTTP mutate → app/platform command → Bus
                                 ├─ slog debug (always)
                                 ├─ audit_events (always, org-scoped; admin API/UI)
                                 └─ platform_event_outbox (if events.outbox_enabled)
worker → ClaimBatch → EventPublisher (log | nats | kafka | noop)
```

## Audit trail vs outbox

| Surface | Purpose | Always on? |
|---------|---------|------------|
| `audit_events` | Org-admin forensics (`GET …/organizations/{orgID}/audit`, Web **Audit**) | Yes |
| `platform_event_outbox` | Cross-process publish to brokers | Only when `events.outbox_enabled` |

The audit bounded context (`internal/audit`) owns entry/record models and the `Store` port. Postgres implements the port; the application layer (`platform.AuditHandler`) maps `platform.Event` → `audit.Entry`. Domains stay free of DB/HTTP/broker tech.

Audit rows include `actor_user_id` (from JWT context via `platform.WithActor`) and a short `summary` string.

## Publish vs emit-only

| Command class | Snapshot publish | Domain event |
|---------------|------------------|--------------|
| Gateway config (keys, providers, routes, quotas, policies, credentials) | yes | yes |
| Tenancy / invites / teams / member roles | no (not in snapshot) | yes |
| Manual `POST /internal/v1/snapshots/publish` | yes | `snapshot.published` |

Membership lifecycle never changes the compiled gateway snapshot, so those commands emit events without republishing.

## Enable

In `configs/local.yaml` (or env):

```yaml
events:
  outbox_enabled: true
  publisher: log   # log | nats | kafka | noop
  nats:
    url: nats://127.0.0.1:4222
    stream: AFI_PLATFORM
    subject_prefix: afi.platform
  kafka:
    brokers: 127.0.0.1:9092
    topic: afi.platform.events
```

Env overrides: `AFI_EVENTS_OUTBOX_ENABLED`, `AFI_EVENTS_PUBLISHER`, `AFI_EVENTS_NATS_URL`, `AFI_EVENTS_KAFKA_BROKERS`, …

Restart **controlplane** (enqueue) and **worker** (drain). With `publisher: log`, the worker logs each published event — useful without a broker.

## Brokers

| Publisher | Adapter | Destination |
|-----------|---------|-------------|
| `log` | `adapters/logpub` | slog info |
| `nats` | `adapters/natsjs` | JetStream subject `afi.platform.{event.name}` |
| `kafka` | `adapters/kafka` | topic (default `afi.platform.events`), key = org id |
| `noop` | — | discard after marking outbox processed |

Optional compose stubs for NATS/Kafka are commented in `docker-compose.yml`.

## Payload

JSON body matches `platform.Event`:

```json
{
  "id": "evt_…",
  "name": "quota.created",
  "resource_id": "quota_…",
  "organization_id": "org_…",
  "actor_user_id": "user_…",
  "at": "2026-07-19T12:00:00Z",
  "meta": {}
}
```

NATS headers: `Nats-Msg-Id`, `afi-event-name`, `afi-organization-id`.  
Kafka headers: `afi-event-id`, `afi-event-name`, `afi-organization-id`.

## Event names

Gateway config: `project.created`, `api_key.*`, `provider.*`, `route.*`, `quota.*`, `policy.*`, `credential.*`, `snapshot.published`.

Tenancy / invites: `org.created`, `member.added`, `member.role_updated`, `invite.created`, `invite.revoked`, `invite.resent`, `invite.accepted`, `team.created`, `team.member_added`, `team.member_role_updated`, `team.member_removed`.

`InviteOrgMember` emits `member.added` when the email already belongs to a user (immediate membership), otherwise `invite.created`.
