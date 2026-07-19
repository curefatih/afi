import { queryOptions } from "@tanstack/react-query";
import { apiFetch } from "#/lib/api-client";
import type { Team } from "#/state/organization-state";

export type TeamMember = {
	user_id: string;
	name: string;
	email: string;
	role: string;
};

export const teamsQueryOptions = (orgId: string) =>
	queryOptions({
		queryKey: ["organizations", orgId, "teams"],
		queryFn: () =>
			apiFetch<Team[]>(`/api/v1/platform/organizations/${orgId}/teams`),
		enabled: !!orgId,
	});

export const teamQueryOptions = (teamId: string) =>
	queryOptions({
		queryKey: ["teams", teamId],
		queryFn: () => apiFetch<Team>(`/api/v1/platform/teams/${teamId}`),
		enabled: !!teamId,
	});

export const teamMembersQueryOptions = (teamId: string) =>
	queryOptions({
		queryKey: ["teams", teamId, "members"],
		queryFn: () =>
			apiFetch<TeamMember[]>(`/api/v1/platform/teams/${teamId}/members`),
		enabled: !!teamId,
	});
