import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@/types';

interface AuthState {
  /** Short-lived access JWT. Memory only — never written to storage. */
  token: string | null;
  user: User | null;
  signIn: (token: string, user: User) => void;
  setToken: (token: string, user?: User) => void;
  signOut: () => void;
}

/**
 * Session model: the 15-minute access token lives only in memory; the
 * long-lived refresh token is an HttpOnly cookie set by the API, so script on
 * the page (including any injected one) can never read it. Only the public
 * profile is persisted, to render the header and to know a session may be
 * restored on the next visit (see api/client restoreSession).
 */
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      signIn: (token, user) => set({ token, user }),
      setToken: (token, user) => set((s) => ({ token, user: user ?? s.user })),
      signOut: () => set({ token: null, user: null }),
    }),
    {
      name: 'co.auth',
      version: 2,
      partialize: (s) => ({ user: s.user }),
      // v1 stored tokens in localStorage: drop them.
      migrate: (persisted) => ({ user: (persisted as { user?: User | null })?.user ?? null }) as AuthState,
    },
  ),
);
