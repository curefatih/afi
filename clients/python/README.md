# afi-platform

Thin synchronous Python client for the AFI control plane (`/api/v1/platform`).

```bash
pip install -e clients/python
```

```python
from afi_platform import PlatformClient

with PlatformClient("http://localhost:8081") as client:
    token = client.login("admin@example.com", "secret")["token"]

client = PlatformClient("http://localhost:8081", get_token=lambda: token)
print(client.list_organizations())
```

Contract: [`../../api/openapi/platform.openapi.yaml`](../../api/openapi/platform.openapi.yaml).
