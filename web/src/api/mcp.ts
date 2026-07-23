import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type MCPBackend = {
	id: string;
	organization_id: string;
	alias: string;
	name: string;
	base_url: string;
	api_key_env: string;
	method_allowlist: string[] | null;
	enabled: boolean;
	created_at: string;
};

export const mcpBackendsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "mcp-backends"],
		queryFn: () =>
			apiFetch<MCPBackend[]>(
				`/api/v1/platform/organizations/${orgId}/mcp-backends`,
			),
		enabled: !!orgId,
	});

export type CreateMCPBackendInput = {
	orgId: string;
	alias: string;
	name: string;
	base_url: string;
	api_key_env?: string;
	method_allowlist?: string[];
	enabled?: boolean;
};

export const createMCPBackendMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateMCPBackendInput) =>
			apiFetch<MCPBackend>(
				`/api/v1/platform/organizations/${orgId}/mcp-backends`,
				{ method: "POST", body },
			),
	});

export type UpdateMCPBackendInput = {
	backendId: string;
	alias?: string;
	name?: string;
	base_url?: string;
	api_key_env?: string;
	method_allowlist?: string[];
	enabled?: boolean;
};

export const updateMCPBackendMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ backendId, ...body }: UpdateMCPBackendInput) =>
			apiFetch<MCPBackend>(`/api/v1/platform/mcp-backends/${backendId}`, {
				method: "PATCH",
				body,
			}),
	});

export const deleteMCPBackendMutationOptions = () =>
	mutationOptions({
		mutationFn: (backendId: string) =>
			apiFetch<void>(`/api/v1/platform/mcp-backends/${backendId}`, {
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

export type TestMCPBackendInput = {
	orgId: string;
	base_url: string;
	api_key_env?: string;
};

export const testMCPBackendMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: TestMCPBackendInput) =>
			apiFetch<ProtocolProbeResult>(
				`/api/v1/platform/organizations/${orgId}/mcp-backends/test`,
				{ method: "POST", body },
			),
	});
