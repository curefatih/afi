import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { PlatformApiError, PlatformClient } from "./client.ts";

describe("PlatformClient", () => {
	it("sends Authorization and parses JSON", async () => {
		const calls: Array<{ url: string; init?: RequestInit }> = [];
		const client = new PlatformClient({
			baseUrl: "http://cp.test",
			getToken: () => "tok",
			fetch: async (url, init) => {
				calls.push({ url: String(url), init });
				return new Response(JSON.stringify({ id: "u1", name: "A", email: "a@b.c", role: "user" }), {
					status: 200,
					headers: { "Content-Type": "application/json" },
				});
			},
		});
		const me = await client.me();
		assert.equal(me.id, "u1");
		assert.equal(calls[0]?.url, "http://cp.test/api/v1/platform/auth/me");
		const headers = new Headers(calls[0]?.init?.headers);
		assert.equal(headers.get("Authorization"), "Bearer tok");
	});

	it("maps error envelope", async () => {
		const client = new PlatformClient({
			baseUrl: "http://cp.test",
			getToken: () => "tok",
			fetch: async () =>
				new Response(JSON.stringify({ error: "nope" }), { status: 403 }),
		});
		await assert.rejects(
			() => client.listOrganizations(),
			(err: unknown) => {
				assert.ok(err instanceof PlatformApiError);
				assert.equal(err.status, 403);
				assert.equal(err.message, "nope");
				return true;
			},
		);
	});

	it("login skips auth header", async () => {
		let auth: string | null = null;
		const client = new PlatformClient({
			baseUrl: "http://cp.test/",
			fetch: async (_url, init) => {
				auth = new Headers(init?.headers).get("Authorization");
				return new Response(JSON.stringify({ token: "jwt" }), { status: 200 });
			},
		});
		const res = await client.login({ email: "a@b.c", password: "x" });
		assert.equal(res.token, "jwt");
		assert.equal(auth, null);
	});
});
