import { useAuthStore } from "#/state/auth-state";
import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ location }) => {
      const isAuthenticated = useAuthStore.getState().isAuthenticated; // might throw on network error
      if (!isAuthenticated) {
        throw redirect({
          to: "/auth/login",
          search: { redirect: location.href },
        });
      }
      return { user: useAuthStore.getState().user };
  },
});

