import { queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type AuditRecord = {
	id: string;
	name: string;
	organization_id: string;
	resource_id: string;
	actor_user_id: string;
	actor_email?: string;
	actor_name?: string;
	summary: string;
	meta?: Record<string, string>;
	at: string;
};

export type AuditFilters = {
	limit?: number;
	from?: string;
	to?: string;
	name?: string;
};

/** Known platform audit event names (keep in sync with platform.EventName). */
export const AUDIT_EVENT_TYPES = [
	{ value: "org.created", label: "Organization created" },
	{ value: "member.added", label: "Member added" },
	{ value: "member.role_updated", label: "Member role updated" },
	{ value: "invite.created", label: "Invite created" },
	{ value: "invite.revoked", label: "Invite revoked" },
	{ value: "invite.resent", label: "Invite resent" },
	{ value: "invite.accepted", label: "Invite accepted" },
	{ value: "team.created", label: "Team created" },
	{ value: "team.member_added", label: "Team member added" },
	{ value: "team.member_role_updated", label: "Team member role updated" },
	{ value: "team.member_removed", label: "Team member removed" },
	{ value: "project.created", label: "Project created" },
	{ value: "api_key.created", label: "API key created" },
	{ value: "api_key.deleted", label: "API key deleted" },
	{ value: "provider.created", label: "Provider created" },
	{ value: "provider.updated", label: "Provider updated" },
	{ value: "provider.deleted", label: "Provider deleted" },
	{ value: "route.created", label: "Route created" },
	{ value: "route.updated", label: "Route updated" },
	{ value: "route.deleted", label: "Route deleted" },
	{ value: "org.default_retry.updated", label: "Default retry updated" },
	{ value: "quota.created", label: "Quota created" },
	{ value: "quota.updated", label: "Quota updated" },
	{ value: "quota.deleted", label: "Quota deleted" },
	{ value: "policy.created", label: "Policy created" },
	{ value: "policy.updated", label: "Policy updated" },
	{ value: "policy.deleted", label: "Policy deleted" },
	{ value: "wasm_hook.created", label: "WASM hook created" },
	{ value: "wasm_hook.updated", label: "WASM hook updated" },
	{ value: "wasm_hook.deleted", label: "WASM hook deleted" },
	{ value: "mcp_backend.created", label: "MCP backend created" },
	{ value: "mcp_backend.updated", label: "MCP backend updated" },
	{ value: "mcp_backend.deleted", label: "MCP backend deleted" },
	{ value: "a2a_agent.created", label: "A2A agent created" },
	{ value: "a2a_agent.updated", label: "A2A agent updated" },
	{ value: "a2a_agent.deleted", label: "A2A agent deleted" },
	{ value: "credential.created", label: "Credential created" },
	{ value: "credential.updated", label: "Credential updated" },
	{ value: "credential.rotated", label: "Credential rotated" },
	{ value: "credential.deleted", label: "Credential deleted" },
	{ value: "credential.assigned", label: "Credential assigned" },
	{ value: "credential.unassigned", label: "Credential unassigned" },
	{ value: "snapshot.published", label: "Snapshot published" },
] as const;

export type AuditEventName = (typeof AUDIT_EVENT_TYPES)[number]["value"];

function auditQueryString(filters: AuditFilters) {
	const params = new URLSearchParams();
	if (filters.limit != null) params.set("limit", String(filters.limit));
	if (filters.from) params.set("from", filters.from);
	if (filters.to) params.set("to", filters.to);
	if (filters.name) params.set("name", filters.name);
	const qs = params.toString();
	return qs ? `?${qs}` : "";
}

export const auditQueryOptions = (orgId: string, filters: AuditFilters = {}) =>
	queryOptions({
		queryKey: ["organizations", orgId, "audit", filters],
		queryFn: () =>
			apiFetch<AuditRecord[]>(
				`/api/v1/platform/organizations/${orgId}/audit${auditQueryString({
					limit: filters.limit ?? 100,
					...filters,
				})}`,
			),
		enabled: !!orgId,
	});
