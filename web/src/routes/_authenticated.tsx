import { useAuthState } from "#/state/auth-state";
import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ location }) => {
      const isAuthenticated = useAuthState.getState().isAuthenticated; // might throw on network error
      if (!isAuthenticated) {
        throw redirect({
          to: "/auth/login",
          search: { redirect: location.href },
        });
      }
      return { isAuthenticated };
  },
});

