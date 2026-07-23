# MCP and A2A (Web UI)

Org owners and admins manage protocol gateway upstreams under **Governance → MCP** and **Governance → A2A**. Changes publish a new gateway snapshot so the data plane can proxy traffic on the next request.

Members can view the lists; only owners and admins can create, edit, delete, or **Test connection**.

## MCP backends

Open **Governance → MCP**.

Each backend maps an **alias** to a remote [Streamable HTTP](https://modelcontextprotocol.io/) MCP server. Clients call the gateway with a virtual API key:

| Method | Gateway path |
|--------|----------------|
| `POST` / `GET` / `DELETE` | `/mcp/{alias}` |

### Fields

| Field | Required | Notes |
|-------|----------|--------|
| **Alias** | yes | URL slug (`docs`, `github`). Becomes `/mcp/{alias}`. |
| **Name** | yes | Display name in the console |
| **Base URL** | yes | Absolute `http(s)` MCP endpoint |
| **API key env** | no | Env var on the gateway process; injected as `Authorization: Bearer …` upstream |
| **Method allowlist** | no | Comma-separated JSON-RPC methods (e.g. `tools/list, tools/call`). Empty allows all. Supports `resources/*`-style prefixes. |
| **Enabled** | — | Disabled backends are omitted from the snapshot route map |

### Test connection

On **Add** or **Edit**, use **Test connection** to probe the base URL from the control plane (MCP `initialize`). Optional API key env is resolved on the control plane host. Success means the upstream responded over HTTP (including auth/protocol 4xx); network failures and 5xx fail the test.

### Try in the playground

Open **Playground → MCP** to list tools and call them (or send raw JSON-RPC) through the gateway. Pick an enabled backend alias, then **List tools** / **Call tool**. The playground uses the local-dev gateway URL and API key (`VITE_GATEWAY_API_*`), same as chat/TTS/STT.

### Example (curl)

1. Create a backend with alias `docs`, base URL `https://mcp.example.com/mcp`, optional `MCP_API_KEY`.
2. Ensure the gateway process has that env var if set.
3. Call:

```bash
curl -s http://localhost:8080/mcp/docs \
  -H "Authorization: Bearer $AFI_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

## A2A agents

Open **Governance → A2A**.

Each agent maps an **alias** to a remote [Agent2Agent](https://a2a-protocol.org/) JSON-RPC endpoint. Discovery and messaging go through the gateway:

| Surface | Gateway path |
|---------|----------------|
| Agent Card | `GET /a2a/{alias}/.well-known/agent-card.json` |
| JSON-RPC | `POST /a2a/{alias}` |

The Agent Card’s endpoint URL is **rewritten** to the gateway so clients do not bypass auth and quotas.

### Fields

| Field | Required | Notes |
|-------|----------|--------|
| **Alias** | yes | URL slug; becomes `/a2a/{alias}` |
| **Name** | yes | Display name |
| **Upstream URL** | yes | Absolute `http(s)` A2A JSON-RPC endpoint |
| **Card URL** | no | Agent Card fetch URL; default `{upstream}/.well-known/agent-card.json` |
| **Card cache JSON** | no | Inline card JSON used instead of fetching when set |
| **API key env** | no | Env var for Bearer injection to upstream / card fetch |
| **Auth scheme** | no | Metadata string (e.g. `bearer`) |
| **Enabled** | — | Disabled agents are not routed |

### Test connection

**Test connection** on Add/Edit GETs the Agent Card (card URL or derived well-known path) from the control plane, with optional API key env. Same success criteria as MCP.

### Example

1. Create an agent with alias `helper` and upstream `https://agent.example/rpc`.
2. Discover:

```bash
curl -s http://localhost:8080/a2a/helper/.well-known/agent-card.json \
  -H "Authorization: Bearer $AFI_API_KEY"
```

3. Send a message (JSON-RPC shape depends on the upstream):

```bash
curl -s http://localhost:8080/a2a/helper \
  -H "Authorization: Bearer $AFI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"text":"hi"}]}}}'
```

## Governance notes

* Creating, updating, or deleting a backend/agent **publishes a snapshot**; the gateway hot-reloads.
* Quotas, CEL policies, hooks (`BeforeCall` / `AfterCall`), and `X-AFI-Tags` apply with modalities `mcp` and `a2a`.
* Credentials today use **API key env** on the gateway (same spirit as LLM providers). BYOK assignment for protocol upstreams is not in the UI yet.

Architecture and pipeline details: [Architecture](../../development/architecture.md). Design depth: repository `internal-docs/mcp-a2a-gateway.md`.

Related: [Web UI overview](../web-ui.md), [Policies](policies.md).
