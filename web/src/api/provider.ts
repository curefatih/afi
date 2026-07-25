import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type ProviderCapabilities = {
	chat: boolean;
	stream: boolean;
	tts?: boolean;
	stt?: boolean;
	embedding?: boolean;
	image?: boolean;
};

export type ProviderHealthStatus = "healthy" | "degraded" | "down" | "unknown";

export type ProviderHealth = {
	provider_id: string;
	name: string;
	type: string;
	requests: number;
	errors: number;
	error_rate: number;
	avg_latency_ms: number;
	status: ProviderHealthStatus;
};

export type Provider = {
	id: string;
	organization_id: string;
	name: string;
	type: string;
	base_url: string;
	api_key_env: string;
	capabilities: ProviderCapabilities;
	created_at: string;
};

export type ProviderTypePreset = {
	type: string;
	name: string;
	base_url: string;
	api_key_env: string;
	auth_mode: string;
	capabilities: ProviderCapabilities;
};

export type ProviderTypePresetMap = Record<
	string,
	{
		name: string;
		base_url: string;
		api_key_env: string;
		caps: ProviderCapabilities;
		auth_mode: string;
	}
>;

export function providerTypesToPresetMap(
	types: ProviderTypePreset[],
): ProviderTypePresetMap {
	const out: ProviderTypePresetMap = {};
	for (const t of types) {
		out[t.type] = {
			name: t.name,
			base_url: t.base_url,
			api_key_env: t.api_key_env,
			caps: t.capabilities,
			auth_mode: t.auth_mode,
		};
	}
	return out;
}

export const providerTypesQueryOptions = () =>
	queryOptions({
		queryKey: ["provider-types"],
		queryFn: () =>
			apiFetch<ProviderTypePreset[]>(`/api/v1/platform/provider-types`),
		staleTime: 60_000,
	});

export const providersQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "providers"],
		queryFn: () =>
			apiFetch<Provider[]>(`/api/v1/platform/organizations/${orgId}/providers`),
		enabled: !!orgId,
	});

export const providerHealthQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "providers", "health"],
		queryFn: () =>
			apiFetch<ProviderHealth[]>(
				`/api/v1/platform/organizations/${orgId}/providers/health`,
			),
		enabled: !!orgId,
		refetchInterval: 30_000,
	});

export type CreateProviderInput = {
	orgId: string;
	name: string;
	type?: string;
	base_url: string;
	api_key_env?: string;
	capabilities?: ProviderCapabilities;
};

export const createProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateProviderInput) =>
			apiFetch<Provider>(`/api/v1/platform/organizations/${orgId}/providers`, {
				method: "POST",
				body,
			}),
	});

export type UpdateProviderInput = {
	providerId: string;
	name: string;
	base_url: string;
	api_key_env: string;
};

export const updateProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ providerId, ...body }: UpdateProviderInput) =>
			apiFetch<Provider>(`/api/v1/platform/providers/${providerId}`, {
				method: "PATCH",
				body,
			}),
	});

export const deleteProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: (providerId: string) =>
			apiFetch<void>(`/api/v1/platform/providers/${providerId}`, {
				method: "DELETE",
			}),
	});
