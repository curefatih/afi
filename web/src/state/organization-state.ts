import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

type Project = {
  id: string;
  team_id: string;
  name: string;
  created_at: string;
  updated_at: string;
};

type Organization = {
  id: string;
  name: string;
  projects: Project[];
  created_at?: string;
};

type AuthState = {
  orgs: Organization[];
  activeOrg?: Organization;
  actions: {
    setOrganizations: (orgs: Organization[]) => void;
    setActiveOrg: (org: Organization) => void;
  };
};

export const useOrgStore = create<AuthState>()(
  persist(
    (set) => ({
      orgs: [],
      actions: {
        setOrganizations: (orgs) =>
          set({
            orgs,
          }),
        setActiveOrg: (org) =>
          set({
            activeOrg: org,
          }),
      },
    }),
    {
      name: "org-state",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        activeOrg: state.activeOrg,
      }),
    },
  ),
);

export const useActiveOrg = () => useOrgStore((state) => state.activeOrg);
