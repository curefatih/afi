import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

type Organization = {
  id: string;
  name: string;
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
      orgs: [
        {
          id: "org_1",
          name: "AFI Inc.",
        },
        {
          id: "org_2",
          name: "Personal",
        },
        {
          id: "org_3",
          name: "AI Research",
        },
      ],
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
