import httpx

from afi_platform import PlatformApiError, PlatformClient


def test_me_sends_bearer():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers.get("Authorization") == "Bearer tok"
        assert request.url.path == "/api/v1/platform/auth/me"
        return httpx.Response(
            200,
            json={"id": "u1", "name": "A", "email": "a@b.c", "role": "user"},
        )

    transport = httpx.MockTransport(handler)
    http = httpx.Client(transport=transport, base_url="http://cp.test")
    client = PlatformClient("http://cp.test", get_token=lambda: "tok", client=http)
    me = client.me()
    assert me["id"] == "u1"
    client.close()


def test_error_envelope():
    transport = httpx.MockTransport(
        lambda request: httpx.Response(403, json={"error": "nope"})
    )
    http = httpx.Client(transport=transport, base_url="http://cp.test")
    client = PlatformClient("http://cp.test", get_token=lambda: "tok", client=http)
    try:
        client.list_organizations()
        assert False, "expected error"
    except PlatformApiError as exc:
        assert exc.status == 403
        assert str(exc) == "nope"
    finally:
        client.close()


def test_login_no_auth():
    seen: dict[str, str | None] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen["auth"] = request.headers.get("Authorization")
        return httpx.Response(200, json={"token": "jwt"})

    transport = httpx.MockTransport(handler)
    http = httpx.Client(transport=transport, base_url="http://cp.test")
    client = PlatformClient("http://cp.test", client=http)
    assert client.login("a@b.c", "x")["token"] == "jwt"
    assert seen["auth"] is None
    client.close()
