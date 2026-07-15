import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { useOrgStore } from "./organization-state";

type User = {
  id: string;
  name: string;
  email: string;
  role: string;
  accessToken: string | null;
  refreshToken: string | null;
};

type AuthState = {
  isAuthenticated: boolean;
  user: User | null;
  actions: {
    setUser: (user: User | null) => void;
    logout: () => void;
  };
};

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      isAuthenticated: false,
      user: null,
      accessToken: null,
      refreshToken: null,
      actions: {
        setUser: (user) =>
          set({
            user,
            isAuthenticated: !!user,
          }),
        logout: () => {
          useOrgStore.setState(useOrgStore.getInitialState());
          set({
            isAuthenticated: false,
            user: null,
          });
        },
      },
    }),
    {
      name: "auth-state",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        isAuthenticated: state.isAuthenticated,
        user: state.user,
        accessToken: state.user?.accessToken,
        refreshToken: state.user?.refreshToken,
      }),
    },
  ),
);

export const useAuthUser = () => useAuthStore((state) => state.user);
export const useIsAuthenticated = () =>
  useAuthStore((state) => state.isAuthenticated);
export const useAuthActions = () => useAuthStore((state) => state.actions);
