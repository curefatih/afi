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
	created_at: string;
};

export const usageQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "usage"],
		queryFn: () =>
			apiFetch<UsageEvent[]>(
				`/api/v1/platform/organizations/${orgId}/usage?limit=50`,
			),
		enabled: !!orgId,
	});
