import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type Team = {
  id: string;
  team_id: string;
  name: string;
  created_at: string;
  updated_at: string;
};

export type Project = {
  id: string;
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
  actions: {
    setOrganizations: (orgs: Organization[]) => void;
    setActiveOrg: (org: Organization) => void;
    setActiveTeamById: (teamId: string) => void;
    addActiveOrganizationTeams: (teams: Team[]) => void;
  };
};

export const useOrgStore = create<OrganizationState>()(
  persist(
    (set, get) => ({
      orgs: [],
      actions: {
        setOrganizations: (orgs) =>
          set({
            orgs,
          }),
        setActiveOrg: (org) => {
          if (org.id === get().activeOrgId) {
            return;
          }
          set({
            activeOrgId: org.id,
          });
        },
        setActiveTeamById(teamId) {
          const exists = get()
            .orgs.find((i) => i.id == get().activeOrgId)
            ?.teams.find((t) => t.id === teamId);

          if (!exists) {
            console.log("no active team found");
            
            return;
          }

          set({
            activeTeamId: teamId,
          });
        },
        addActiveOrganizationTeams(teams: Team[]) {
          if (!get().activeOrgId) {
            return;
          }

          set({
            orgs: get().orgs.map((org) => {
              if (org.id === get().activeOrgId) {
                return {
                  ...org,
                  teams,
                };
              }
              return org;
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
      }),
    },
  ),
);

export const useActiveOrg = () =>
  useOrgStore((state) => state.orgs.find((o) => o.id === state.activeOrgId));
export const useActiveTeam = () =>
  useOrgStore((state) => {
    const orgId = state.activeOrgId;
    const org = state.orgs.find((o) => o.id === orgId);
    if (!org) return undefined;

    return org.teams?.find((p) => p.id === state.activeTeamId);
  });
