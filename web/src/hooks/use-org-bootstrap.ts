import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import {
	organizationsQueryOptions,
	orgProjectsQueryOptions,
	orgTeamsQueryOptions,
	toOrganization,
} from "#/api/organization";
import { useIsAuthenticated } from "#/state/auth-state";
import { useOrgActions, useOrgStore } from "#/state/organization-state";

/** Loads orgs and hydrates teams/projects for the active organization. */
export function useOrgBootstrap() {
	const isAuthenticated = useIsAuthenticated();
	const actions = useOrgActions();
	const activeOrgId = useOrgStore((s) => s.activeOrgId);

	const orgsQuery = useQuery({
		...organizationsQueryOptions(),
		enabled: isAuthenticated,
	});

	useEffect(() => {
		if (!orgsQuery.data) return;
		actions.setOrganizations(
			orgsQuery.data.map((org) =>
				toOrganization(
					org,
					useOrgStore.getState().orgs.find((o) => o.id === org.id)?.teams ?? [],
					useOrgStore.getState().orgs.find((o) => o.id === org.id)?.projects ??
						[],
				),
			),
		);
	}, [orgsQuery.data, actions]);

	const resolvedOrgId = activeOrgId || orgsQuery.data?.[0]?.id || undefined;

	const teamsQuery = useQuery({
		...orgTeamsQueryOptions(resolvedOrgId ?? ""),
		enabled: isAuthenticated && !!resolvedOrgId,
	});

	const projectsQuery = useQuery({
		...orgProjectsQueryOptions(resolvedOrgId ?? ""),
		enabled: isAuthenticated && !!resolvedOrgId,
	});

	useEffect(() => {
		if (!resolvedOrgId || !teamsQuery.data) return;
		actions.setOrgTeams(resolvedOrgId, teamsQuery.data);
	}, [resolvedOrgId, teamsQuery.data, actions]);

	useEffect(() => {
		if (!resolvedOrgId || !projectsQuery.data) return;
		actions.setOrgProjects(resolvedOrgId, projectsQuery.data);
	}, [resolvedOrgId, projectsQuery.data, actions]);

	const isBootstrapping =
		orgsQuery.isPending ||
		(!!resolvedOrgId && (teamsQuery.isPending || projectsQuery.isPending));

	const isError =
		orgsQuery.isError || teamsQuery.isError || projectsQuery.isError;

	const error =
		orgsQuery.error || teamsQuery.error || projectsQuery.error || null;

	return {
		orgsQuery,
		teamsQuery,
		projectsQuery,
		activeOrgId: resolvedOrgId,
		isBootstrapping,
		isError,
		error,
		refetch: () => {
			void orgsQuery.refetch();
			void teamsQuery.refetch();
			void projectsQuery.refetch();
		},
	};
}
