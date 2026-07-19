import { queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type RequestPolicy = {
	id: string;
	organization_id: string;
	name: string;
	expression: string;
	enabled: boolean;
	priority: number;
	created_at: string;
};

export const policiesQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "policies"],
		queryFn: () =>
			apiFetch<RequestPolicy[]>(
				`/api/v1/platform/organizations/${orgId}/policies`,
			),
		enabled: Boolean(orgId),
	});

export function createPolicyMutationOptions() {
	return {
		mutationFn: (input: {
			orgId: string;
			name: string;
			expression: string;
			enabled?: boolean;
			priority?: number;
		}) =>
			apiFetch<RequestPolicy>(
				`/api/v1/platform/organizations/${input.orgId}/policies`,
				{
					method: "POST",
					body: JSON.stringify({
						name: input.name,
						expression: input.expression,
						enabled: input.enabled ?? true,
						priority: input.priority ?? 100,
					}),
				},
			),
	};
}

export function deletePolicyMutationOptions() {
	return {
		mutationFn: (policyId: string) =>
			apiFetch<void>(`/api/v1/platform/policies/${policyId}`, {
				method: "DELETE",
			}),
	};
}
