from __future__ import annotations

import base64

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import Encoding, NoEncryption, PrivateFormat

from afi_platform.httpsign import REQUIRED_COMPONENTS, content_digest_sha256, sign_headers


def _pem() -> tuple[bytes, Ed25519PrivateKey]:
    key = Ed25519PrivateKey.generate()
    pem = key.private_bytes(Encoding.PEM, PrivateFormat.PKCS8, NoEncryption())
    return pem, key


def test_content_digest_shape() -> None:
    d = content_digest_sha256(b"hello")
    assert d.startswith("sha-256=:")
    assert d.endswith(":")


def test_sign_headers_roundtrip_verify_base() -> None:
    pem, key = _pem()
    body = b'{"ping":true}'
    headers = sign_headers(
        method="POST",
        url="http://localhost:8080/v1/chat/completions",
        body=body,
        private_key_pem=pem,
        key_id="local-signer",
        nonce="n1",
        created=1_700_000_000,
    )
    assert "Content-Digest" in headers
    assert headers["Signature-Input"].startswith("sig1=")
    assert headers["Signature"].startswith("sig1=:")
    for c in REQUIRED_COMPONENTS:
        assert c in headers["Signature-Input"]

    # Recompute signature base and verify with the same key material.
    digest = headers["Content-Digest"]
    sig_params = headers["Signature-Input"].removeprefix("sig1=")
    sig_base = "\n".join(
        [
            '"@method": POST',
            '"@path": /v1/chat/completions',
            '"@query": ?',
            f'"content-digest": {digest}',
            f'"@signature-params": {sig_params}',
        ]
    ).encode()
    raw = base64.b64decode(headers["Signature"].removeprefix("sig1=:").removesuffix(":"))
    key.public_key().verify(raw, sig_base)


def test_sign_headers_with_query() -> None:
    pem, key = _pem()
    headers = sign_headers(
        method="GET",
        url="/v1/models?foo=1",
        body=b"",
        private_key_pem=pem,
        key_id="kid",
        nonce="n",
        created=1,
    )
    assert "@query" in headers["Signature-Input"]
    digest = headers["Content-Digest"]
    sig_params = headers["Signature-Input"].removeprefix("sig1=")
    sig_base = "\n".join(
        [
            '"@method": GET',
            '"@path": /v1/models',
            '"@query": ?foo=1',
            f'"content-digest": {digest}',
            f'"@signature-params": {sig_params}',
        ]
    ).encode()
    raw = base64.b64decode(headers["Signature"].removeprefix("sig1=:").removesuffix(":"))
    key.public_key().verify(raw, sig_base)
