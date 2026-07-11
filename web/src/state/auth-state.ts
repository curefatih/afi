import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'

type User = {
  id: string
  name: string
  email: string
  role: string
}

type AuthState = {
  isAuthenticated: boolean
  setIsAuthenticated: (isAuthenticated: boolean) => void
  user: User | null
  setUser: (user: User | null) => void
  logout: () => void
}

export const useAuthState = create<AuthState>()(persist((set) => ({
  isAuthenticated: false,
  user: null,
  setIsAuthenticated: (isAuthenticated: boolean) => set({ isAuthenticated }),
  setUser: (user: User | null) => set({ user, isAuthenticated: user !== null && user !== undefined }),
  logout: () => set({ isAuthenticated: false, user: null }),
}), {
  name: 'auth-state',
  storage: createJSONStorage(() => localStorage),
}))