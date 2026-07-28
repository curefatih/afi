import assert from "node:assert/strict";
import { createPublicKey, verify as cryptoVerify } from "node:crypto";
import { describe, it } from "node:test";
import {
	REQUIRED_COMPONENTS,
	contentDigestSha256,
	generateEd25519Pem,
	pathAndQuery,
	signHeaders,
} from "./httpsign.ts";

describe("httpsign", () => {
	it("pathAndQuery handles absolute URLs and paths", () => {
		assert.deepEqual(pathAndQuery("http://localhost:8080/v1/chat/completions"), {
			path: "/v1/chat/completions",
			query: "?",
		});
		assert.deepEqual(pathAndQuery("/v1/models?foo=1"), {
			path: "/v1/models",
			query: "?foo=1",
		});
	});

	it("contentDigestSha256 shape", () => {
		const d = contentDigestSha256("hello");
		assert.ok(d.startsWith("sha-256=:"));
		assert.ok(d.endsWith(":"));
	});

	it("signHeaders produces a verifiable signature", () => {
		const { privateKeyPem, publicKeyPem } = generateEd25519Pem();
		const body = '{"ping":true}';
		const headers = signHeaders({
			method: "POST",
			url: "http://localhost:8080/v1/chat/completions",
			body,
			privateKeyPem,
			keyId: "local-signer",
			nonce: "n1",
			created: 1_700_000_000,
		});
		assert.ok(headers["Signature-Input"]?.startsWith("sig1="));
		assert.ok(headers.Signature?.startsWith("sig1=:"));
		for (const c of REQUIRED_COMPONENTS) {
			assert.ok(headers["Signature-Input"]?.includes(c));
		}

		const digest = headers["Content-Digest"]!;
		const sigParams = headers["Signature-Input"]!.slice("sig1=".length);
		const sigBase = [
			`"@method": POST`,
			`"@path": /v1/chat/completions`,
			`"@query": ?`,
			`"content-digest": ${digest}`,
			`"@signature-params": ${sigParams}`,
		].join("\n");
		const raw = Buffer.from(
			headers.Signature!.slice("sig1=:".length, -1),
			"base64",
		);
		const ok = cryptoVerify(
			null,
			Buffer.from(sigBase, "utf8"),
			createPublicKey(publicKeyPem),
			raw,
		);
		assert.equal(ok, true);
	});
});
