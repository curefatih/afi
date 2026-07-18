import { useAuthStore } from "#/state/auth-state";
import { useOrgStore } from "#/state/organization-state";
import { mutationOptions, queryOptions } from "@tanstack/react-query";

export const teamsQueryOptions = () =>
  mutationOptions({
    mutationFn: async () => {
      const isAuthenticated = useAuthStore.getState().isAuthenticated;
      if (!isAuthenticated) {
        throw new Error("Not authenticated");
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
      if (!res.ok) throw new Error("Failed to fetch teams");
      const teams = await res.json();

      return teams.map((i) => ({ ...i, previewMembers: [] }));
    },
  });

export const teamQueryOptions = (teamId: string) =>
  queryOptions({
    queryKey: ["team", teamId],
    queryFn: async () => {
      const isAuthenticated = useAuthStore.getState().isAuthenticated;
      if (!isAuthenticated) {
        throw new Error("Not authenticated");
      }

      const res = await fetch(
        `http://localhost:8080/api/v1/platform/teams/${teamId}`,
        {
          headers: {
            Authorization: `Bearer ${useAuthStore.getState().user?.accessToken}`,
          },
        },
      );
      if (!res.ok) throw new Error("Failed to fetch teams");
      const team = await res.json();

      return team;
    },
  });

export const teamMembersQueryOptions = (teamId: string) =>
  queryOptions({
    queryKey: ["teamMembers", teamId],
    queryFn: async () => {
      const isAuthenticated = useAuthStore.getState().isAuthenticated;
      if (!isAuthenticated) {
        throw new Error("Not authenticated");
      }

      const res = await fetch(
        `http://localhost:8080/api/v1/platform/teams/${teamId}/members`,
        {
          headers: {
            Authorization: `Bearer ${useAuthStore.getState().user?.accessToken}`,
          },
        },
      );
      if (!res.ok) throw new Error("Failed to fetch team members");
      const team = await res.json();

      return team;
    },
  });
