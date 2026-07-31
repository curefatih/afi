import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type Region = {
	id: string;
	slug: string;
	name: string;
	status: string;
	created_at: string;
	updated_at: string;
};

export type GatewayDeployment = {
	id: string;
	region_id: string;
	name: string;
	public_base_url?: string;
	status: string;
	last_seen_at?: string | null;
	reported_snapshot_version: number;
	reported_build?: string;
	created_at: string;
	updated_at: string;
};

export type DeploymentWithToken = {
	deployment: GatewayDeployment;
	join_token: string;
};

export const regionsQueryOptions = () =>
	queryOptions({
		queryKey: ["regions"],
		queryFn: () => apiFetch<Region[]>("/api/v1/platform/regions"),
	});

export const regionQueryOptions = (regionId: string) =>
	queryOptions({
		queryKey: ["regions", regionId],
		queryFn: () =>
			apiFetch<Region>(`/api/v1/platform/regions/${regionId}`),
		enabled: !!regionId,
	});

export const deploymentsQueryOptions = (regionId: string) =>
	queryOptions({
		queryKey: ["regions", regionId, "deployments"],
		queryFn: () =>
			apiFetch<GatewayDeployment[]>(
				`/api/v1/platform/regions/${regionId}/deployments`,
			),
		enabled: !!regionId,
		refetchInterval: 15_000,
	});

export const createRegionMutationOptions = () =>
	mutationOptions({
		mutationFn: (body: { slug: string; name: string }) =>
			apiFetch<Region>("/api/v1/platform/regions", {
				method: "POST",
				body,
			}),
	});

export const updateRegionMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			regionId,
			...body
		}: {
			regionId: string;
			name?: string;
			status?: string;
		}) =>
			apiFetch<Region>(`/api/v1/platform/regions/${regionId}`, {
				method: "PATCH",
				body,
			}),
	});

export const registerDeploymentMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			regionId,
			...body
		}: {
			regionId: string;
			name: string;
			public_base_url?: string;
		}) =>
			apiFetch<DeploymentWithToken>(
				`/api/v1/platform/regions/${regionId}/deployments`,
				{ method: "POST", body },
			),
	});

export const rotateJoinTokenMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			regionId,
			deploymentId,
		}: {
			regionId: string;
			deploymentId: string;
		}) =>
			apiFetch<DeploymentWithToken>(
				`/api/v1/platform/regions/${regionId}/deployments/${deploymentId}/rotate-join-token`,
				{ method: "POST" },
			),
	});

export type OrgRegionMembership = {
	organization_id: string;
	region_id: string;
	status: string;
	created_at: string;
	updated_at: string;
};

export type RegionConfigOverlay = {
	organization_id: string;
	region_id: string;
	payload: Record<string, unknown>;
	created_at: string;
	updated_at: string;
};

export const regionMembershipsQueryOptions = (regionId: string) =>
	queryOptions({
		queryKey: ["regions", regionId, "organizations"],
		queryFn: () =>
			apiFetch<OrgRegionMembership[]>(
				`/api/v1/platform/regions/${regionId}/organizations`,
			),
		enabled: !!regionId,
	});

export const regionOverlayQueryOptions = (regionId: string, orgId: string) =>
	queryOptions({
		queryKey: ["regions", regionId, "organizations", orgId, "overlay"],
		queryFn: () =>
			apiFetch<RegionConfigOverlay>(
				`/api/v1/platform/regions/${regionId}/organizations/${orgId}/overlay`,
			),
		enabled: !!regionId && !!orgId,
		retry: false,
	});

export const bindOrgMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			regionId,
			organization_id,
			status,
		}: {
			regionId: string;
			organization_id: string;
			status?: string;
		}) =>
			apiFetch<OrgRegionMembership>(
				`/api/v1/platform/regions/${regionId}/organizations`,
				{
					method: "POST",
					body: { organization_id, status: status ?? "active" },
				},
			),
	});

export const unbindOrgMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			regionId,
			orgId,
		}: {
			regionId: string;
			orgId: string;
		}) =>
			apiFetch<void>(
				`/api/v1/platform/regions/${regionId}/organizations/${orgId}`,
				{ method: "DELETE" },
			),
	});

export const putOverlayMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			regionId,
			orgId,
			payload,
		}: {
			regionId: string;
			orgId: string;
			payload: Record<string, unknown>;
		}) =>
			apiFetch<RegionConfigOverlay>(
				`/api/v1/platform/regions/${regionId}/organizations/${orgId}/overlay`,
				{ method: "PUT", body: payload },
			),
	});

export const deleteOverlayMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			regionId,
			orgId,
		}: {
			regionId: string;
			orgId: string;
		}) =>
			apiFetch<void>(
				`/api/v1/platform/regions/${regionId}/organizations/${orgId}/overlay`,
				{ method: "DELETE" },
			),
	});
