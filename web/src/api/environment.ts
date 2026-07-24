import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";

export type Environment = {
	id: string;
	organization_id: string;
	project_id: string;
	name: string;
	slug: string;
	created_at: string;
};

export const environmentsQueryOptions = (orgId: string, projectId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "projects", projectId, "environments"],
		queryFn: () =>
			apiFetch<Environment[]>(
				`/api/v1/platform/organizations/${orgId}/projects/${projectId}/environments`,
			),
		enabled: !!orgId && !!projectId,
	});

export type CreateEnvironmentInput = {
	orgId: string;
	projectId: string;
	name: string;
	slug: string;
};

export const createEnvironmentMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, projectId, name, slug }: CreateEnvironmentInput) =>
			apiFetch<Environment>(
				`/api/v1/platform/organizations/${orgId}/projects/${projectId}/environments`,
				{ method: "POST", body: { name, slug } },
			),
	});

export const deleteEnvironmentMutationOptions = () =>
	mutationOptions({
		mutationFn: (environmentId: string) =>
			apiFetch<void>(`/api/v1/platform/environments/${environmentId}`, {
				method: "DELETE",
			}),
	});
