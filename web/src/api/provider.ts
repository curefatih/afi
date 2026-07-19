import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type Provider = {
	id: string;
	organization_id: string;
	name: string;
	type: string;
	base_url: string;
	api_key_env: string;
	created_at: string;
};

export const providersQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "providers"],
		queryFn: () =>
			apiFetch<Provider[]>(
				`/api/v1/platform/organizations/${orgId}/providers`,
			),
		enabled: !!orgId,
	});

export type CreateProviderInput = {
	orgId: string;
	name: string;
	type?: string;
	base_url: string;
	api_key_env?: string;
};

export const createProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateProviderInput) =>
			apiFetch<Provider>(
				`/api/v1/platform/organizations/${orgId}/providers`,
				{ method: "POST", body },
			),
	});

export const deleteProviderMutationOptions = () =>
	mutationOptions({
		mutationFn: (providerId: string) =>
			apiFetch<void>(`/api/v1/platform/providers/${providerId}`, {
				method: "DELETE",
			}),
	});
