import { useAuthStore } from "#/state/auth-state";
import { useOrgStore } from "#/state/organization-state";
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

      useOrgStore.getState().actions.setOrganizations(orgs);
      if (orgs.length > 0) {
        useOrgStore.getState().actions.setActiveOrg(orgs[0]);
      }
      return orgs;
    },
  });
