/**
 * RFC 9421 gateway request signing for AFI signed-request auth.
 */
import {
	createHash,
	createPrivateKey,
	generateKeyPairSync,
	randomBytes,
	sign as cryptoSign,
} from "node:crypto";

export const SIGNATURE_NAME = "sig1";
export const REQUIRED_COMPONENTS = [
	"@method",
	"@path",
	"@query",
	"content-digest",
] as const;

export type SignHeadersInput = {
	method: string;
	/** Absolute URL or path (with optional query). */
	url: string;
	body?: Uint8Array | string;
	privateKeyPem: string;
	keyId: string;
	nonce?: string;
	created?: number;
};

function toBytes(body: Uint8Array | string | undefined): Buffer {
	if (body == null) return Buffer.alloc(0);
	if (typeof body === "string") return Buffer.from(body, "utf8");
	return Buffer.from(body);
}

/** RFC 9421 @path / @query from a URL or path string. Empty query → "?". */
export function pathAndQuery(urlOrPath: string): { path: string; query: string } {
	if (/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(urlOrPath)) {
		const u = new URL(urlOrPath);
		return { path: u.pathname || "/", query: `?${u.search.startsWith("?") ? u.search.slice(1) : u.search}` };
	}
	const q = urlOrPath.indexOf("?");
	if (q < 0) return { path: urlOrPath || "/", query: "?" };
	return { path: urlOrPath.slice(0, q) || "/", query: `?${urlOrPath.slice(q + 1)}` };
}

export function contentDigestSha256(body: Uint8Array | string = ""): string {
	const digest = createHash("sha256").update(toBytes(body)).digest("base64");
	return `sha-256=:${digest}:`;
}

/**
 * Build Content-Digest / Signature-Input / Signature headers for a gateway call.
 * Covers @method, @path, @query, and content-digest (gateway-required set).
 */
export function signHeaders(input: SignHeadersInput): Record<string, string> {
	const { method, url, privateKeyPem, keyId } = input;
	if (!keyId) throw new Error("keyId is required");
	const body = toBytes(input.body);
	const { path, query } = pathAndQuery(url);
	const created = input.created ?? Math.floor(Date.now() / 1000);
	const nonce = input.nonce ?? randomBytes(16).toString("hex");
	const digest = contentDigestSha256(body);
	const sigParams =
		`("@method" "@path" "@query" "content-digest")` +
		`;created=${created};nonce="${nonce}";alg="ed25519";keyid="${keyId}"`;
	const sigBase = [
		`"@method": ${method.toUpperCase()}`,
		`"@path": ${path}`,
		`"@query": ${query}`,
		`"content-digest": ${digest}`,
		`"@signature-params": ${sigParams}`,
	].join("\n");

	const key = createPrivateKey(privateKeyPem);
	const signature = cryptoSign(null, Buffer.from(sigBase, "utf8"), key);

	return {
		"Content-Digest": digest,
		"Signature-Input": `${SIGNATURE_NAME}=${sigParams}`,
		Signature: `${SIGNATURE_NAME}=:${signature.toString("base64")}:`,
	};
}

export function mergeSignedHeaders(
	headers: Record<string, string> | undefined,
	input: SignHeadersInput,
): Record<string, string> {
	return { ...(headers ?? {}), ...signHeaders(input) };
}

/** Generate an Ed25519 PKCS#8 / SPKI PEM key pair (tests and local setup). */
export function generateEd25519Pem(): {
	privateKeyPem: string;
	publicKeyPem: string;
} {
	const { privateKey, publicKey } = generateKeyPairSync("ed25519");
	return {
		privateKeyPem: privateKey.export({ type: "pkcs8", format: "pem" }).toString(),
		publicKeyPem: publicKey.export({ type: "spki", format: "pem" }).toString(),
	};
}
