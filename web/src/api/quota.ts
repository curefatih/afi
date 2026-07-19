import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type Quota = {
	id: string;
	organization_id: string;
	scope_type: string;
	scope_id: string;
	metric: string;
	limit_value: number;
	window: string;
	created_at: string;
};

export const quotasQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "quotas"],
		queryFn: () =>
			apiFetch<Quota[]>(`/api/v1/platform/organizations/${orgId}/quotas`),
		enabled: !!orgId,
	});

export type CreateQuotaInput = {
	orgId: string;
	scope_type: string;
	scope_id: string;
	metric: string;
	limit_value: number;
	window?: string;
};

export const createQuotaMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateQuotaInput) =>
			apiFetch<Quota>(`/api/v1/platform/organizations/${orgId}/quotas`, {
				method: "POST",
				body,
			}),
	});

export const deleteQuotaMutationOptions = () =>
	mutationOptions({
		mutationFn: (quotaId: string) =>
			apiFetch<void>(`/api/v1/platform/quotas/${quotaId}`, {
				method: "DELETE",
			}),
	});
