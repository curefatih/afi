import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type Team = {
	id: string;
	team_id: string;
	organization_id?: string;
	name: string;
	created_at: string;
	updated_at: string;
};

export type Project = {
	id: string;
	organization_id?: string;
	team_id: string;
	name: string;
	created_at: string;
	updated_at: string;
};

export type Organization = {
	id: string;
	name: string;
	projects: Project[];
	teams: Team[];
	created_at?: string;
};

type OrganizationState = {
	orgs: Organization[];
	activeOrgId?: string;
	activeTeamId?: string;
	activeProjectId?: string;
	actions: {
		setOrganizations: (orgs: Organization[]) => void;
		setActiveOrgById: (orgId: string) => void;
		setActiveTeamById: (teamId: string | undefined) => void;
		setActiveProjectById: (projectId: string | undefined) => void;
		setOrgTeams: (orgId: string, teams: Team[]) => void;
		setOrgProjects: (orgId: string, projects: Project[]) => void;
		upsertTeam: (orgId: string, team: Team) => void;
		upsertProject: (orgId: string, project: Project) => void;
	};
};

export const useOrgStore = create<OrganizationState>()(
	persist(
		(set, get) => ({
			orgs: [],
			actions: {
				setOrganizations: (orgs) => {
					const { activeOrgId } = get();
					const nextActive =
						(activeOrgId && orgs.find((o) => o.id === activeOrgId)?.id) ||
						orgs[0]?.id;
					set({
						orgs,
						activeOrgId: nextActive,
					});
				},
				setActiveOrgById: (orgId) => {
					const org = get().orgs.find((o) => o.id === orgId);
					if (!org) return;
					set({
						activeOrgId: orgId,
						activeTeamId: undefined,
						activeProjectId: undefined,
					});
				},
				setActiveTeamById(teamId) {
					if (!teamId) {
						set({ activeTeamId: undefined });
						return;
					}
					const org = get().orgs.find((o) => o.id === get().activeOrgId);
					if (!org?.teams.find((t) => t.id === teamId)) return;
					set({ activeTeamId: teamId });
				},
				setActiveProjectById(projectId) {
					set({ activeProjectId: projectId });
				},
				setOrgTeams(orgId, teams) {
					set({
						orgs: get().orgs.map((org) =>
							org.id === orgId ? { ...org, teams } : org,
						),
					});
					const { activeTeamId } = get();
					if (
						activeTeamId &&
						!teams.some((t) => t.id === activeTeamId) &&
						get().activeOrgId === orgId
					) {
						set({ activeTeamId: teams[0]?.id });
					} else if (
						!activeTeamId &&
						teams.length > 0 &&
						get().activeOrgId === orgId
					) {
						set({ activeTeamId: teams[0].id });
					}
				},
				setOrgProjects(orgId, projects) {
					set({
						orgs: get().orgs.map((org) =>
							org.id === orgId ? { ...org, projects } : org,
						),
					});
				},
				upsertTeam(orgId, team) {
					set({
						orgs: get().orgs.map((org) => {
							if (org.id !== orgId) return org;
							const exists = org.teams.some((t) => t.id === team.id);
							return {
								...org,
								teams: exists
									? org.teams.map((t) => (t.id === team.id ? team : t))
									: [...org.teams, team],
							};
						}),
					});
					const { activeTeamId, activeOrgId } = get();
					if (!activeTeamId && activeOrgId === orgId) {
						set({ activeTeamId: team.id });
					}
				},
				upsertProject(orgId, project) {
					set({
						orgs: get().orgs.map((org) => {
							if (org.id !== orgId) return org;
							const exists = org.projects.some((p) => p.id === project.id);
							return {
								...org,
								projects: exists
									? org.projects.map((p) => (p.id === project.id ? project : p))
									: [...org.projects, project],
							};
						}),
					});
				},
			},
		}),
		{
			name: "org-state",
			storage: createJSONStorage(() => localStorage),
			partialize: (state) => ({
				activeOrgId: state.activeOrgId,
				activeTeamId: state.activeTeamId,
				activeProjectId: state.activeProjectId,
			}),
		},
	),
);

export const useActiveOrg = () =>
	useOrgStore((state) => state.orgs.find((o) => o.id === state.activeOrgId));

export const useActiveTeam = () =>
	useOrgStore((state) => {
		const org = state.orgs.find((o) => o.id === state.activeOrgId);
		if (!org) return undefined;
		return org.teams?.find((t) => t.id === state.activeTeamId);
	});

export const useActiveProject = () =>
	useOrgStore((state) => {
		const org = state.orgs.find((o) => o.id === state.activeOrgId);
		if (!org) return undefined;
		return org.projects?.find((p) => p.id === state.activeProjectId);
	});

export const useOrgActions = () => useOrgStore((state) => state.actions);
