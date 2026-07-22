import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type PolicyActionType =
	| "allow"
	| "deny"
	| "set_header"
	| "use_credential";

/** @deprecated Use PolicyActionType */
export type PolicyAction = PolicyActionType;

export type PolicyActionConfig = {
	header?: string;
	value?: string;
	value_expr?: string;
	credential_name?: string;
	credential_name_expr?: string;
};

export type PolicyThen = {
	type: PolicyActionType;
	config?: PolicyActionConfig;
};

export type RequestPolicy = {
	id: string;
	organization_id: string;
	name: string;
	expression: string;
	actions: PolicyThen[];
	/** @deprecated Prefer actions */
	action?: PolicyActionType;
	/** @deprecated Prefer actions */
	action_config?: PolicyActionConfig;
	enabled: boolean;
	priority: number;
	created_at: string;
};

/** Normalize API policy to always expose actions[]. */
export function policyActions(p: RequestPolicy): PolicyThen[] {
	if (Array.isArray(p.actions) && p.actions.length > 0) {
		return p.actions.map((a) => ({
			type: (a.type || "deny") as PolicyActionType,
			config: a.config ?? {},
		}));
	}
	if (p.action) {
		return [{ type: p.action, config: p.action_config ?? {} }];
	}
	return [{ type: "deny", config: {} }];
}

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
	actions: PolicyThen[];
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
						actions: body.actions,
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
	actions?: PolicyThen[];
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
