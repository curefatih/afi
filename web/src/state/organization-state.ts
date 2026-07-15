import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

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
  created_at?: string;
};

type OrganizationState = {
  orgs: Organization[];
  activeOrgId?: string;
  activeProjectId?: string;
  actions: {
    setOrganizations: (orgs: Organization[]) => void;
    setActiveOrg: (org: Organization) => void;
    setActiveProjectByID: (projectID: string) => void;
    addActiveOrganizationProjects: (projects: Project[]) => void;
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
        setActiveProjectByID(projectID) {
          const exists = get()
            .orgs.find((i) => i.id == get().activeOrgId)
            ?.projects.find((p) => p.id === projectID);

          if (!exists) {
            return;
          }

          set({
            activeProjectId: projectID,
          });
        },
        addActiveOrganizationProjects(projects: Project[]) {
          if (!get().activeOrgId) {
            return;
          }

          set({
            orgs: get().orgs.map((org) => {
              if (org.id === get().activeOrgId) {
                return {
                  ...org,
                  projects,
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
        activeProjectId: state.activeProjectId,
      }),
    },
  ),
);

export const useActiveOrg = () =>
  useOrgStore((state) => state.orgs.find((o) => o.id === state.activeOrgId));
export const useActiveProject = () =>
  useOrgStore((state) => {
    const orgId = state.activeOrgId;
    const org = state.orgs.find((o) => o.id === orgId);
    if (!org) return undefined;

    return org.projects?.find((p) => p.id === state.activeProjectId);
  });
