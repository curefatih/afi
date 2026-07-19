import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type RouteConfig = {
	id: string;
	organization_id: string;
	model: string;
	provider_id: string;
	target_model: string;
	created_at: string;
};

export const routesQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "routes"],
		queryFn: () =>
			apiFetch<RouteConfig[]>(
				`/api/v1/platform/organizations/${orgId}/routes`,
			),
		enabled: !!orgId,
	});

export type CreateRouteInput = {
	orgId: string;
	model: string;
	provider_id: string;
	target_model?: string;
};

export const createRouteMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateRouteInput) =>
			apiFetch<RouteConfig>(
				`/api/v1/platform/organizations/${orgId}/routes`,
				{ method: "POST", body },
			),
	});

export const deleteRouteMutationOptions = () =>
	mutationOptions({
		mutationFn: (routeId: string) =>
			apiFetch<void>(`/api/v1/platform/routes/${routeId}`, {
				method: "DELETE",
			}),
	});
