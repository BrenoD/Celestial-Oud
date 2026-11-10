import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@/types';

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  signIn: (token: string, user: User, refreshToken?: string | null) => void;
  setTokens: (token: string, refreshToken?: string | null) => void;
  signOut: () => void;
}

/**
 * JWT + user kept in localStorage. The token is sent as a Bearer header by
 * api/client; a 401 from the API clears it.
 */
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,
      signIn: (token, user, refreshToken = null) => set({ token, user, refreshToken }),
      setTokens: (token, refreshToken = null) => set((s) => ({ token, refreshToken: refreshToken ?? s.refreshToken })),
      signOut: () => set({ token: null, user: null, refreshToken: null }),
    }),
    { name: 'co.auth', version: 1 },
  ),
);
