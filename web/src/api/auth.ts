import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";
import { useAuthStore } from "#/state/auth-state";

export type MeResponse = {
	id: string;
	name: string;
	email: string;
	role: string;
};

export const meQueryOptions = () =>
	queryOptions({
		queryKey: ["me"],
		queryFn: () => apiFetch<MeResponse>("/api/v1/platform/auth/me"),
	});

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
		},
	});
