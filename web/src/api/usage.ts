import { queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type UsageEvent = {
	id: number;
	organization_id: string;
	project_id: string;
	api_key_id: string;
	model: string;
	status: string;
	latency_ms: number;
	prompt_tokens: number;
	completion_tokens: number;
	modality: string;
	metrics?: Record<string, unknown>;
	cost_usd?: number | null;
	created_at: string;
	key_name?: string;
	key_kind?: string;
	owner_user_id?: string;
	owner_email?: string;
	owner_name?: string;
};

export type UsageSummaryBucket = {
	bucket: string;
	label: string;
	requests: number;
	cost_usd: number;
	prompt_tokens: number;
	completion_tokens: number;
	metrics_totals?: Record<string, number>;
	key_kind?: string;
	owner_email?: string;
	owner_name?: string;
};

export type UsageFilters = {
	limit?: number;
	project_id?: string;
	api_key_id?: string;
	model?: string;
	modality?: string;
	from?: string;
	to?: string;
};

export type UsageGroupBy = "day" | "model" | "key" | "modality";

function usageQueryString(filters: UsageFilters & { group_by?: UsageGroupBy }) {
	const params = new URLSearchParams();
	if (filters.limit != null) params.set("limit", String(filters.limit));
	if (filters.project_id) params.set("project_id", filters.project_id);
	if (filters.api_key_id) params.set("api_key_id", filters.api_key_id);
	if (filters.model) params.set("model", filters.model);
	if (filters.modality) params.set("modality", filters.modality);
	if (filters.from) params.set("from", filters.from);
	if (filters.to) params.set("to", filters.to);
	if (filters.group_by) params.set("group_by", filters.group_by);
	const qs = params.toString();
	return qs ? `?${qs}` : "";
}

export const usageQueryOptions = (orgId: string, filters: UsageFilters = {}) =>
	queryOptions({
		queryKey: ["organizations", orgId, "usage", filters],
		queryFn: () =>
			apiFetch<UsageEvent[]>(
				`/api/v1/platform/organizations/${orgId}/usage${usageQueryString({
					limit: filters.limit ?? 50,
					...filters,
				})}`,
			),
		enabled: !!orgId,
	});

export const usageSummaryQueryOptions = (
	orgId: string,
	groupBy: UsageGroupBy,
	filters: UsageFilters = {},
) =>
	queryOptions({
		queryKey: ["organizations", orgId, "usage", "summary", groupBy, filters],
		queryFn: () =>
			apiFetch<UsageSummaryBucket[]>(
				`/api/v1/platform/organizations/${orgId}/usage/summary${usageQueryString(
					{ ...filters, group_by: groupBy },
				)}`,
			),
		enabled: !!orgId,
	});

export function formatUsageQuantity(e: UsageEvent): string {
	const m = e.metrics ?? {};
	if (e.modality === "tts" && typeof m.characters === "number") {
		return `${m.characters} chars`;
	}
	if (e.modality === "stt" && typeof m.audio_seconds === "number") {
		return `${m.audio_seconds}s audio`;
	}
	if (typeof m.images === "number") {
		return `${m.images} image${m.images === 1 ? "" : "s"}`;
	}
	if (e.prompt_tokens > 0 || e.completion_tokens > 0) {
		return `${e.prompt_tokens}/${e.completion_tokens} tok`;
	}
	return "—";
}

export function formatUsageOwner(e: UsageEvent): string {
	if (e.key_kind === "service_account") {
		return "Service account";
	}
	if (e.owner_name || e.owner_email) {
		return e.owner_name || e.owner_email || "—";
	}
	return "—";
}
