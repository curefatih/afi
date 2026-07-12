import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { useAuthStore } from "#/state/auth-state";

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
      const res = await fetch("http://localhost:8080/api/v1/platform/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(data),
      });
      if (!res.ok) throw new Error("Failed to login");
      return res.json();
      },
    onSuccess: async (data) => {
      // Fetch the user info after receiving the token.
      const token = data.token;
      // Save token to store as accessToken, clear refreshToken.
      // You may want to call /me or a user info route to populate current user.
      const res = await fetch("http://localhost:8080/api/v1/platform/auth/me", {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (!res.ok) {
        throw new Error("Failed to fetch user after login");
      }
      const user = await res.json();
      if (!user) {
        throw new Error("Failed to fetch user after login");
      }
      useAuthStore.getState().actions.setUser({
        accessToken: token,
        refreshToken: null,
        id: user.id,
        name: user.name,
        email: user.email,
        role: user.role,
      });
    },
  });