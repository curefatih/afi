"""Thin synchronous HTTP client for the AFI Platform API."""

from __future__ import annotations

from typing import Any, Callable, Mapping, MutableMapping, Optional
from urllib.parse import urlencode, urljoin

import httpx

TokenGetter = Callable[[], Optional[str]]


class PlatformApiError(Exception):
    def __init__(self, message: str, status: int, body: Any = None) -> None:
        super().__init__(message)
        self.status = status
        self.body = body


class PlatformClient:
    """Minimal client for ``/api/v1/platform/*``.

    Parameters
    ----------
    base_url:
        Control plane origin, e.g. ``http://localhost:8081``.
    get_token:
        Optional callable returning a JWT for authenticated requests.
    client:
        Optional pre-configured ``httpx.Client``.
    """

    def __init__(
        self,
        base_url: str,
        *,
        get_token: Optional[TokenGetter] = None,
        client: Optional[httpx.Client] = None,
        timeout: float = 30.0,
    ) -> None:
        self.base_url = base_url.rstrip("/") + "/"
        self._get_token = get_token
        self._owns_client = client is None
        self._client = client or httpx.Client(timeout=timeout)

    def close(self) -> None:
        if self._owns_client:
            self._client.close()

    def __enter__(self) -> "PlatformClient":
        return self

    def __exit__(self, *args: object) -> None:
        self.close()

    def request(
        self,
        method: str,
        path: str,
        *,
        body: Any = None,
        auth: bool = True,
        query: Optional[Mapping[str, Any]] = None,
    ) -> Any:
        headers: MutableMapping[str, str] = {}
        if body is not None:
            headers["Content-Type"] = "application/json"
        if auth:
            token = self._get_token() if self._get_token else None
            if not token:
                raise PlatformApiError("missing access token", 401)
            headers["Authorization"] = f"Bearer {token}"
        params = None
        if query:
            params = {k: v for k, v in query.items() if v is not None}
        url = urljoin(self.base_url, path.lstrip("/"))
        if params:
            url = f"{url}?{urlencode(params)}"
        res = self._client.request(
            method,
            url,
            headers=headers,
            json=body if body is not None else None,
        )
        if res.status_code == 204:
            return None
        parsed: Any
        try:
            parsed = res.json()
        except Exception:
            parsed = res.text
        if res.is_error:
            message = res.reason_phrase or "request failed"
            if isinstance(parsed, dict) and isinstance(parsed.get("error"), str):
                message = parsed["error"]
            raise PlatformApiError(message, res.status_code, parsed)
        return parsed

    def healthz(self) -> dict[str, Any]:
        return self.request("GET", "/healthz", auth=False)

    def login(self, email: str, password: str) -> dict[str, Any]:
        return self.request(
            "POST",
            "/api/v1/platform/auth/login",
            body={"email": email, "password": password},
            auth=False,
        )

    def auth_features(self) -> dict[str, Any]:
        return self.request("GET", "/api/v1/platform/auth/features", auth=False)

    def register(self, email: str, name: str, password: str) -> dict[str, Any]:
        return self.request(
            "POST",
            "/api/v1/platform/auth/register",
            body={"email": email, "name": name, "password": password},
            auth=False,
        )

    def request_password_reset(self, email: str) -> dict[str, Any]:
        return self.request(
            "POST",
            "/api/v1/platform/auth/password-reset",
            body={"email": email},
            auth=False,
        )

    def confirm_password_reset(self, token: str, password: str) -> dict[str, Any]:
        return self.request(
            "POST",
            f"/api/v1/platform/auth/password-reset/{token}",
            body={"password": password},
            auth=False,
        )

    def me(self) -> dict[str, Any]:
        return self.request("GET", "/api/v1/platform/auth/me")

    def list_organizations(self) -> list[Any]:
        return self.request("GET", "/api/v1/platform/organizations")

    def create_organization(self, name: str) -> dict[str, Any]:
        return self.request(
            "POST", "/api/v1/platform/organizations", body={"name": name}
        )

    def list_org_keys(self, org_id: str) -> list[Any]:
        return self.request("GET", f"/api/v1/platform/organizations/{org_id}/keys")

    def create_org_key(self, org_id: str, body: Mapping[str, Any]) -> dict[str, Any]:
        return self.request(
            "POST",
            f"/api/v1/platform/organizations/{org_id}/keys",
            body=dict(body),
        )

    def list_providers(self, org_id: str) -> list[Any]:
        return self.request(
            "GET", f"/api/v1/platform/organizations/{org_id}/providers"
        )

    def list_routes(self, org_id: str) -> list[Any]:
        return self.request("GET", f"/api/v1/platform/organizations/{org_id}/routes")

    def list_usage(
        self, org_id: str, query: Optional[Mapping[str, Any]] = None
    ) -> list[Any]:
        return self.request(
            "GET",
            f"/api/v1/platform/organizations/{org_id}/usage",
            query=query,
        )

    def list_audit(
        self, org_id: str, query: Optional[Mapping[str, Any]] = None
    ) -> list[Any]:
        return self.request(
            "GET",
            f"/api/v1/platform/organizations/{org_id}/audit",
            query=query,
        )
