import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "#/state/auth-state";
import { type ApiError, apiFetch } from "./api-client";

describe("apiFetch", () => {
	beforeEach(() => {
		useAuthStore.setState({
			isAuthenticated: true,
			user: {
				id: "u1",
				name: "Admin",
				email: "admin@afi.local",
				role: "admin",
				accessToken: "tok_test",
				refreshToken: null,
			},
		});
	});

	afterEach(() => {
		vi.unstubAllGlobals();
		useAuthStore.setState({
			isAuthenticated: false,
			user: null,
		});
	});

	it("sends bearer auth and parses JSON", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ ok: true }), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			}),
		);
		vi.stubGlobal("fetch", fetchMock);

		const data = await apiFetch<{ ok: boolean }>("/v1/me");
		expect(data.ok).toBe(true);
		expect(fetchMock).toHaveBeenCalledOnce();
		const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
		expect(url).toContain("/v1/me");
		expect(new Headers(init.headers).get("Authorization")).toBe(
			"Bearer tok_test",
		);
	});

	it("throws ApiError with server message", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: "nope" }), {
					status: 403,
					statusText: "Forbidden",
					headers: { "Content-Type": "application/json" },
				}),
			),
		);

		await expect(apiFetch("/v1/secret")).rejects.toMatchObject({
			name: "ApiError",
			status: 403,
			message: "nope",
		} satisfies Partial<ApiError>);
	});

	it("requires auth token when auth is enabled", async () => {
		useAuthStore.setState({ isAuthenticated: false, user: null });
		await expect(apiFetch("/v1/me")).rejects.toMatchObject({
			status: 401,
			message: "Not authenticated",
		});
	});

	it("returns undefined for 204", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
		);
		await expect(apiFetch("/v1/noop")).resolves.toBeUndefined();
	});
});
