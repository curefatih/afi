"""RFC 9421 gateway request signing helpers for AFI signed-request auth."""

from __future__ import annotations

import base64
import hashlib
import secrets
import time
from typing import Mapping
from urllib.parse import urlsplit

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import load_pem_private_key

SIGNATURE_NAME = "sig1"
REQUIRED_COMPONENTS = ("@method", "@path", "@query", "content-digest")


def _load_private_key(private_key_pem: str | bytes) -> Ed25519PrivateKey:
    raw = private_key_pem.encode() if isinstance(private_key_pem, str) else private_key_pem
    key = load_pem_private_key(raw, password=None)
    if not isinstance(key, Ed25519PrivateKey):
        raise TypeError("private key must be Ed25519")
    return key


def content_digest_sha256(body: bytes) -> str:
    """Build an RFC 9530 Content-Digest header value for sha-256."""
    digest = base64.b64encode(hashlib.sha256(body).digest()).decode("ascii")
    return f"sha-256=:{digest}:"


def _path_and_query(url_or_path: str) -> tuple[str, str]:
    parts = urlsplit(url_or_path)
    path = parts.path or "/"
    # RFC 9421 @query is "?" + raw query (even when empty → "?").
    query = "?" + (parts.query or "")
    return path, query


def sign_headers(
    *,
    method: str,
    url: str,
    body: bytes = b"",
    private_key_pem: str | bytes,
    key_id: str,
    nonce: str | None = None,
    created: int | None = None,
) -> dict[str, str]:
    """Return Content-Digest / Signature-Input / Signature headers for a gateway call.

    Covers ``@method``, ``@path``, ``@query``, and ``content-digest`` — the set
    required by the AFI gateway.
    """
    if not key_id:
        raise ValueError("key_id is required")
    body = body or b""
    path, query = _path_and_query(url)
    created_ts = int(time.time()) if created is None else int(created)
    nonce_val = nonce or secrets.token_hex(16)
    digest = content_digest_sha256(body)
    sig_params = (
        '("@method" "@path" "@query" "content-digest")'
        f';created={created_ts};nonce="{nonce_val}";alg="ed25519";keyid="{key_id}"'
    )
    sig_base = "\n".join(
        [
            f'"@method": {method.upper()}',
            f'"@path": {path}',
            f'"@query": {query}',
            f'"content-digest": {digest}',
            f'"@signature-params": {sig_params}',
        ]
    ).encode("utf-8")
    signature = _load_private_key(private_key_pem).sign(sig_base)
    return {
        "Content-Digest": digest,
        "Signature-Input": f"{SIGNATURE_NAME}={sig_params}",
        "Signature": f"{SIGNATURE_NAME}=:{base64.b64encode(signature).decode('ascii')}:",
    }


def merge_signed_headers(
    headers: Mapping[str, str] | None,
    *,
    method: str,
    url: str,
    body: bytes = b"",
    private_key_pem: str | bytes,
    key_id: str,
    nonce: str | None = None,
) -> dict[str, str]:
    """Copy ``headers`` and overlay RFC 9421 signing headers."""
    out = dict(headers or {})
    out.update(
        sign_headers(
            method=method,
            url=url,
            body=body,
            private_key_pem=private_key_pem,
            key_id=key_id,
            nonce=nonce,
        )
    )
    return out
