import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";
import type { Project } from "#/state/organization-state";

export const projectsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "projects"],
		queryFn: () =>
			apiFetch<Project[]>(`/api/v1/platform/organizations/${orgId}/projects`),
		enabled: !!orgId,
	});

export type CreateProjectInput = {
	orgId: string;
	name: string;
	team_id: string;
};

export const createProjectMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, name, team_id }: CreateProjectInput) =>
			apiFetch<Project>(`/api/v1/platform/organizations/${orgId}/projects`, {
				method: "POST",
				body: { name, team_id },
			}),
	});
