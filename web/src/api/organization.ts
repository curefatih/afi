import { queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";
import type { Organization, Project, Team } from "#/state/organization-state";

export type OrgSummary = {
	id: string;
	name: string;
	created_at?: string;
};

export const organizationsQueryOptions = () =>
	queryOptions({
		queryKey: ["organizations"],
		queryFn: () => apiFetch<OrgSummary[]>("/api/v1/platform/organizations"),
	});

export const orgTeamsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "teams"],
		queryFn: () =>
			apiFetch<Team[]>(`/api/v1/platform/organizations/${orgId}/teams`),
		enabled: !!orgId,
	});

export const orgProjectsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "projects"],
		queryFn: () =>
			apiFetch<Project[]>(`/api/v1/platform/organizations/${orgId}/projects`),
		enabled: !!orgId,
	});

export function toOrganization(
	org: OrgSummary,
	teams: Team[] = [],
	projects: Project[] = [],
): Organization {
	return {
		id: org.id,
		name: org.name,
		created_at: org.created_at,
		teams,
		projects,
	};
}
