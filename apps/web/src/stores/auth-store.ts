import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api, type User } from "@/lib/api/client";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;

  // Actions
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, nickname: string) => Promise<void>;
  logout: () => void;
  fetchProfile: () => Promise<void>;
  setTokens: (accessToken: string, refreshToken: string) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (email: string, password: string) => {
        set({ isLoading: true });
        try {
          const { data } = await api.login({ email, password });
          api.setAccessToken(data.access_token);
          localStorage.setItem("refresh_token", data.refresh_token);
          set({ user: data.user, isAuthenticated: true, isLoading: false });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      register: async (email: string, password: string, nickname: string) => {
        set({ isLoading: true });
        try {
          const { data } = await api.register({ email, password, nickname });
          api.setAccessToken(data.access_token);
          localStorage.setItem("refresh_token", data.refresh_token);
          set({ user: data.user, isAuthenticated: true, isLoading: false });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      logout: () => {
        api.setAccessToken(null);
        localStorage.removeItem("refresh_token");
        set({ user: null, isAuthenticated: false });
      },

      fetchProfile: async () => {
        try {
          const { data } = await api.getProfile();
          set({ user: data, isAuthenticated: true });
        } catch {
          get().logout();
        }
      },

      setTokens: (accessToken: string, refreshToken: string) => {
        api.setAccessToken(accessToken);
        localStorage.setItem("refresh_token", refreshToken);
      },
    }),
    {
      name: "auth-storage",
      partialize: (state) => ({ user: state.user, isAuthenticated: state.isAuthenticated }),
    }
  )
);
