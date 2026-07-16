import { useAuthStore } from "#/state/auth-state";
import { useOrgStore, type Organization } from "#/state/organization-state";
import { mutationOptions } from "@tanstack/react-query";

export const organizationsQueryOptions = () =>
  mutationOptions({
    mutationFn: async () => {
      const isAuthenticated = useAuthStore.getState().isAuthenticated;
      if (!isAuthenticated) {
        throw new Error("Not authenticated");
      }
      const res = await fetch(
        "http://localhost:8080/api/v1/platform/organizations",
        {
          headers: {
            Authorization: `Bearer ${useAuthStore.getState().user?.accessToken}`,
          },
        },
      );
      if (!res.ok) throw new Error("Failed to fetch organizations");
      const orgs = await res.json();

      useOrgStore
        .getState()
        .actions.setOrganizations(
          orgs.map((org: Organization) => ({
            ...org,
            projects: [],
            teams: [],
          })),
        );
      if (orgs.length > 0) {
        useOrgStore.getState().actions.setActiveOrg(orgs[0]);
      }
      return orgs;
    },
    onSuccess: async (data) => {
      if (!data || !data.length) {
        console.log(data);
        return;
      }

      const activeOrgId = useOrgStore.getState().activeOrgId;
      if (!activeOrgId) {
        return;
      }

      const res = await fetch(
        `http://localhost:8080/api/v1/platform/organizations/${activeOrgId}/teams`,
        {
          headers: {
            Authorization: `Bearer ${useAuthStore.getState().user?.accessToken}`,
          },
        },
      );
      if (!res.ok) throw new Error("Failed to fetch organizations");
      const teams = await res.json();

      useOrgStore.getState().actions.addActiveOrganizationTeams(teams);

      return teams;
    },
  });
