import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type RoutingStrategy = "ordered" | "weighted";

export type RouteFallback = {
	provider_id: string;
	target_model: string;
	weight?: number;
};

export type BackoffConfig = {
	strategy: "fixed" | "exponential";
	base_delay: string;
	max_delay?: string;
	multiplier?: number;
};

export type RetryConfig = {
	max_attempts: number;
	backoff: BackoffConfig;
};

export type RouteConfig = {
	id: string;
	organization_id: string;
	model: string;
	provider_id: string;
	target_model: string;
	fallbacks: RouteFallback[];
	retry?: RetryConfig | null;
	routing_strategy?: RoutingStrategy;
	weight?: number;
	created_at: string;
};

export const routesQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "routes"],
		queryFn: () =>
			apiFetch<RouteConfig[]>(`/api/v1/platform/organizations/${orgId}/routes`),
		enabled: !!orgId,
	});

export type CreateRouteInput = {
	orgId: string;
	model: string;
	provider_id: string;
	target_model?: string;
	fallbacks?: RouteFallback[];
	retry?: RetryConfig | null;
	routing_strategy?: RoutingStrategy;
	weight?: number;
};

export const createRouteMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, ...body }: CreateRouteInput) =>
			apiFetch<RouteConfig>(`/api/v1/platform/organizations/${orgId}/routes`, {
				method: "POST",
				body,
			}),
	});

export type UpdateRouteInput = {
	routeId: string;
	model: string;
	provider_id: string;
	target_model?: string;
	fallbacks?: RouteFallback[];
	retry?: RetryConfig | null;
	routing_strategy?: RoutingStrategy;
	weight?: number;
};

export const updateRouteMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ routeId, ...body }: UpdateRouteInput) =>
			apiFetch<RouteConfig>(`/api/v1/platform/routes/${routeId}`, {
				method: "PATCH",
				body,
			}),
	});

export const deleteRouteMutationOptions = () =>
	mutationOptions({
		mutationFn: (routeId: string) =>
			apiFetch<void>(`/api/v1/platform/routes/${routeId}`, {
				method: "DELETE",
			}),
	});
