import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { PLATFORM_API_URL } from "#/lib/api-base";
import { apiFetch } from "#/lib/api-client";
import { useAuthStore } from "#/state/auth-state";

export type MeResponse = {
	id: string;
	name: string;
	email: string;
	role: string;
};

export type SSOProvider = {
	id: string;
	display_name: string;
	type: string;
};

export type AuthFeatures = {
	signup_enabled: boolean;
	password_reset_enabled: boolean;
};

export const meQueryOptions = () =>
	queryOptions({
		queryKey: ["me"],
		queryFn: () => apiFetch<MeResponse>("/api/v1/platform/auth/me"),
	});

export const authFeaturesQueryOptions = () =>
	queryOptions({
		queryKey: ["auth-features"],
		queryFn: () =>
			apiFetch<AuthFeatures>("/api/v1/platform/auth/features", { auth: false }),
	});

export const ssoProvidersQueryOptions = () =>
	queryOptions({
		queryKey: ["sso-providers"],
		queryFn: async () => {
			const res = await apiFetch<{ providers: SSOProvider[] }>(
				"/api/v1/platform/auth/sso/providers",
				{ auth: false },
			);
			return res.providers ?? [];
		},
	});

/** Full-page redirect to begin OAuth/OIDC login for a provider. */
export function startSSO(providerID: string, redirect?: string) {
	const url = new URL(
		`${PLATFORM_API_URL}/api/v1/platform/auth/sso/${encodeURIComponent(providerID)}/start`,
	);
	if (redirect) {
		url.searchParams.set("redirect", redirect);
	}
	window.location.assign(url.toString());
}

export async function bootstrapSessionFromToken(token: string) {
	const user = await apiFetch<MeResponse>("/api/v1/platform/auth/me", {
		headers: { Authorization: `Bearer ${token}` },
		auth: false,
	});
	useAuthStore.getState().actions.setUser({
		accessToken: token,
		refreshToken: null,
		id: user.id,
		name: user.name,
		email: user.email,
		role: user.role,
	});
	return user;
}

type LoginRequest = {
	email: string;
	password: string;
};

export const loginMutationOptions = () =>
	mutationOptions({
		mutationFn: async (data: LoginRequest) => {
			const { token } = await apiFetch<{ token: string }>(
				"/api/v1/platform/auth/login",
				{
					method: "POST",
					body: data,
					auth: false,
				},
			);

			return bootstrapSessionFromToken(token);
		},
	});

type RegisterRequest = {
	email: string;
	name: string;
	password: string;
};

export const registerMutationOptions = () =>
	mutationOptions({
		mutationFn: async (data: RegisterRequest) => {
			const { token } = await apiFetch<{ token: string }>(
				"/api/v1/platform/auth/register",
				{
					method: "POST",
					body: data,
					auth: false,
				},
			);
			return bootstrapSessionFromToken(token);
		},
	});

export const requestPasswordResetMutationOptions = () =>
	mutationOptions({
		mutationFn: async (email: string) => {
			return apiFetch<{ ok: boolean }>("/api/v1/platform/auth/password-reset", {
				method: "POST",
				body: { email },
				auth: false,
			});
		},
	});

export const confirmPasswordResetMutationOptions = () =>
	mutationOptions({
		mutationFn: async (data: { token: string; password: string }) => {
			const { token } = await apiFetch<{ token: string }>(
				`/api/v1/platform/auth/password-reset/${encodeURIComponent(data.token)}`,
				{
					method: "POST",
					body: { password: data.password },
					auth: false,
				},
			);
			return bootstrapSessionFromToken(token);
		},
	});
