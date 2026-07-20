import { mutationOptions, queryOptions } from "@tanstack/react-query";
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

export type CreatePolicyInput = {
	orgId: string;
	name: string;
	expression: string;
	enabled?: boolean;
	priority?: number;
};

export const createPolicyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreatePolicyInput) =>
			apiFetch<RequestPolicy>(
				`/api/v1/platform/organizations/${orgId}/policies`,
				{
					method: "POST",
					body: {
						name: body.name,
						expression: body.expression,
						enabled: body.enabled ?? true,
						priority: body.priority ?? 100,
					},
				},
			),
	});

export type UpdatePolicyInput = {
	policyId: string;
	name?: string;
	expression?: string;
	enabled?: boolean;
	priority?: number;
};

export const updatePolicyMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ policyId, ...body }: UpdatePolicyInput) =>
			apiFetch<RequestPolicy>(`/api/v1/platform/policies/${policyId}`, {
				method: "PATCH",
				body,
			}),
	});

export const deletePolicyMutationOptions = () =>
	mutationOptions({
		mutationFn: (policyId: string) =>
			apiFetch<void>(`/api/v1/platform/policies/${policyId}`, {
				method: "DELETE",
			}),
	});

export type ReorderPoliciesInput = {
	orgId: string;
	policies: Array<Pick<RequestPolicy, "id" | "priority">>;
};

/** Persist priority changes in one transactional reorder request. */
export const reorderPoliciesMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, policies }: ReorderPoliciesInput) =>
			apiFetch<RequestPolicy[]>(
				`/api/v1/platform/organizations/${orgId}/policies/reorder`,
				{
					method: "POST",
					body: { items: policies },
				},
			),
	});
