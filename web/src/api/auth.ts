import { useAuthStore } from "#/state/auth-state";
import { mutationOptions, queryOptions } from "@tanstack/react-query";

export const meQueryOptions = () =>
  queryOptions({
    queryKey: ["me"],
    queryFn: async () => {
      const isAuthenticated = useAuthStore.getState().isAuthenticated;
      if (!isAuthenticated) {
        throw new Error("Not authenticated");
      }
      const res = await fetch("http://localhost:8080/api/v1/platform/auth/me", {
        headers: {
          Authorization: `Bearer ${useAuthStore.getState().user?.accessToken}`,
        },
      });
      if (!res.ok) throw new Error("Failed to fetch me");
      return res.json();
    },
  });

type LoginRequest = {
  email: string;
  password: string;
};


export const loginMutationOptions = () =>
  mutationOptions({
    mutationFn: async (data: LoginRequest) => {
      const res = await fetch(
        "http://localhost:8080/api/v1/platform/auth/login",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(data),
        },
      );
      if (!res.ok) throw new Error("Failed to login");
      return res.json();
    },
    onSuccess: async (data) => {
      const token = data.token;

      const res = await fetch("http://localhost:8080/api/v1/platform/auth/me", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Failed to fetch user after login");

      const user = await res.json();

      // Update Zustand state (Updates are synchronous)
      useAuthStore.getState().actions.setUser({
        accessToken: token,
        refreshToken: null,
        id: user.id,
        name: user.name,
        email: user.email,
        role: user.role,
      });

      // Return data so the component can safely handle navigation on resolve
      return user;
    },
  });
