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

## Gateway signed requests

```python
from afi_platform import sign_headers

headers = {
    "Content-Type": "application/json",
    **sign_headers(
        method="POST",
        url="http://localhost:8080/v1/chat/completions",
        body=body,
        private_key_pem=open("signer.pem", "rb").read(),
        key_id="local-signer",
    ),
}
```

Release time: Jul 29 20:57 AM