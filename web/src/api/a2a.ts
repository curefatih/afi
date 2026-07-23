import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type A2AAgent = {
	id: string;
	organization_id: string;
	alias: string;
	name: string;
	upstream_url: string;
	card_url: string;
	card_cache?: unknown;
	api_key_env: string;
	auth_scheme: string;
	enabled: boolean;
	created_at: string;
};

export const a2aAgentsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "a2a-agents"],
		queryFn: () =>
			apiFetch<A2AAgent[]>(
				`/api/v1/platform/organizations/${orgId}/a2a-agents`,
			),
		enabled: !!orgId,
	});

export type CreateA2AAgentInput = {
	orgId: string;
	alias: string;
	name: string;
	upstream_url: string;
	card_url?: string;
	card_cache?: unknown;
	api_key_env?: string;
	auth_scheme?: string;
	enabled?: boolean;
};

export const createA2AAgentMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateA2AAgentInput) =>
			apiFetch<A2AAgent>(`/api/v1/platform/organizations/${orgId}/a2a-agents`, {
				method: "POST",
				body,
			}),
	});

export type UpdateA2AAgentInput = {
	agentId: string;
	alias?: string;
	name?: string;
	upstream_url?: string;
	card_url?: string;
	card_cache?: unknown;
	api_key_env?: string;
	auth_scheme?: string;
	enabled?: boolean;
};

export const updateA2AAgentMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ agentId, ...body }: UpdateA2AAgentInput) =>
			apiFetch<A2AAgent>(`/api/v1/platform/a2a-agents/${agentId}`, {
				method: "PATCH",
				body,
			}),
	});

export const deleteA2AAgentMutationOptions = () =>
	mutationOptions({
		mutationFn: (agentId: string) =>
			apiFetch<void>(`/api/v1/platform/a2a-agents/${agentId}`, {
				method: "DELETE",
			}),
	});

export type ProtocolProbeResult = {
	ok: boolean;
	status_code?: number;
	latency_ms: number;
	error?: string;
	detail?: string;
};

export type TestA2AAgentInput = {
	orgId: string;
	upstream_url: string;
	card_url?: string;
	api_key_env?: string;
};

export const testA2AAgentMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: TestA2AAgentInput) =>
			apiFetch<ProtocolProbeResult>(
				`/api/v1/platform/organizations/${orgId}/a2a-agents/test`,
				{ method: "POST", body },
			),
	});
