import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "#/state/auth-state";
import {
	__resetSessionRedirectForTests,
	type ApiError,
	apiFetch,
} from "./api-client";

describe("apiFetch", () => {
	beforeEach(() => {
		__resetSessionRedirectForTests();
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
		vi.restoreAllMocks();
		__resetSessionRedirectForTests();
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
		const assign = vi.fn();
		vi.stubGlobal("location", {
			...window.location,
			pathname: "/app/dashboard",
			search: "",
			hash: "",
			href: "http://localhost/app/dashboard",
			origin: "http://localhost",
			assign,
		});
		useAuthStore.setState({ isAuthenticated: false, user: null });
		await expect(apiFetch("/v1/me")).rejects.toMatchObject({
			status: 401,
			message: "Not authenticated",
		});
		expect(useAuthStore.getState().isAuthenticated).toBe(false);
		expect(assign).toHaveBeenCalledWith(
			"/auth/login?redirect=%2Fapp%2Fdashboard",
		);
	});

	it("logs out and redirects to login on authenticated 401", async () => {
		const assign = vi.fn();
		vi.stubGlobal("location", {
			...window.location,
			pathname: "/app/organizations",
			search: "",
			hash: "",
			href: "http://localhost/app/organizations",
			origin: "http://localhost",
			assign,
		});
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: "unauthorized" }), {
					status: 401,
					statusText: "Unauthorized",
					headers: { "Content-Type": "application/json" },
				}),
			),
		);

		await expect(apiFetch("/v1/me")).rejects.toMatchObject({
			status: 401,
			message: "unauthorized",
		} satisfies Partial<ApiError>);

		expect(useAuthStore.getState().user).toBeNull();
		expect(useAuthStore.getState().isAuthenticated).toBe(false);
		expect(assign).toHaveBeenCalledWith(
			"/auth/login?redirect=%2Fapp%2Forganizations",
		);
	});

	it("does not redirect on 401 when auth is disabled", async () => {
		const assign = vi.fn();
		vi.stubGlobal("location", {
			...window.location,
			pathname: "/auth/login",
			href: "http://localhost/auth/login",
			origin: "http://localhost",
			assign,
		});
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: "invalid credentials" }), {
					status: 401,
					statusText: "Unauthorized",
					headers: { "Content-Type": "application/json" },
				}),
			),
		);

		await expect(
			apiFetch("/api/v1/platform/auth/login", {
				method: "POST",
				body: { email: "a@b.c", password: "x" },
				auth: false,
			}),
		).rejects.toMatchObject({ status: 401 });

		expect(assign).not.toHaveBeenCalled();
	});

	it("returns undefined for 204", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
		);
		await expect(apiFetch("/v1/noop")).resolves.toBeUndefined();
	});
});
