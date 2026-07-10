## Configuration Specification

### `config/gateway.yaml`

The gateway is configured locally via structural serialization parameters targeting fallback catch-all routing structures.

```yaml
# Local Developer Configuration
api_keys:
  - raw_key_prefix: "sk-project-local-dev-token-12345"
    type: "SERVICE_ACCOUNT"
    project_id: "proj_local"

upstream_credentials:
  - project_id: "proj_local"
    provider: "openai"
    env_var_key: "LOCAL_OPENAI_API_KEY"

routing_rules:
  - id: "rule_1"
    name: "Catch-all Local Dev Rule"
    priority: 1
    is_active: true
    conditions: [] 
    target:
      provider: "openai"
      target_model: "gpt-4o"

```

## Local Deployment Checklist

1. **Expose Environment Variables:** Ensure matching provider keys match selection vault rules inside the local shell:

```bash
export LOCAL_OPENAI_API_KEY="sk-proj-YOUR-ACTUAL-KEY"

```

2. **Execute Gateway Execution Run-loop:**

```bash
go run cmd/gateway/main.go

```

3. **Dispatch Validation Stream Target Verification:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Test packet submission."}],
    "stream": true
  }'

```
