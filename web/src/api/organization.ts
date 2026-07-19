import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";
import type { Organization, Project, Team } from "#/state/organization-state";

export type OrgSummary = {
	id: string;
	name: string;
	created_at?: string;
};

export type OrgMember = {
	user_id: string;
	email: string;
	name: string;
	role: string;
};

export const organizationsQueryOptions = () =>
	queryOptions({
		queryKey: ["organizations"],
		queryFn: () => apiFetch<OrgSummary[]>("/api/v1/platform/organizations"),
	});

export const orgMembersQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "members"],
		queryFn: () =>
			apiFetch<OrgMember[]>(`/api/v1/platform/organizations/${orgId}/members`),
		enabled: !!orgId,
	});

export const createOrganizationMutationOptions = () =>
	mutationOptions({
		mutationFn: (body: { name: string }) =>
			apiFetch<OrgSummary>("/api/v1/platform/organizations", {
				method: "POST",
				body,
			}),
	});

export const addOrgMemberMutationOptions = () =>
	mutationOptions({
		mutationFn: ({ orgId, email }: { orgId: string; email: string }) =>
			apiFetch<OrgMember>(`/api/v1/platform/organizations/${orgId}/members`, {
				method: "POST",
				body: { email },
			}),
	});

export type OrgRole = "owner" | "admin" | "member";

export const updateOrgMemberRoleMutationOptions = () =>
	mutationOptions({
		mutationFn: ({
			orgId,
			userId,
			role,
		}: {
			orgId: string;
			userId: string;
			role: OrgRole;
		}) =>
			apiFetch<OrgMember>(
				`/api/v1/platform/organizations/${orgId}/members/${userId}`,
				{ method: "PATCH", body: { role } },
			),
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
