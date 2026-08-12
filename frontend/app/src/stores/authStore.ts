import { create } from 'zustand';

import type { TokenPair } from '@/api/auth';

const ACCESS_TOKEN_KEY = 'drawo.access_token';
const REFRESH_TOKEN_KEY = 'drawo.refresh_token';
// expiresAt is the absolute timestamp (ms since epoch) at which the current
// access token becomes invalid. It is derived from the server's `expires_in`
// field at the moment tokens are set (login or refresh).
// We persist it in sessionStorage (not localStorage) because:
//  - localStorage survives browser close; a hard expiry wall-clock time
//    persisted across days would be misleading.
//  - sessionStorage matches the tab lifetime, which is what our in-memory
//    state also represents.
const EXPIRES_AT_KEY = 'drawo.expires_at';

type AuthState = {
  accessToken: string | null;
  refreshToken: string | null;
  /** Millisecond epoch timestamp when accessToken expires; null if unknown. */
  expiresAt: number | null;

  /**
   * Persist a fresh token pair (from login or refresh).
   *
   * @param tokens - The full TokenPair from the backend. expires_in is in
   *   seconds; we convert it to an absolute epoch ms timestamp for easier
   *   arithmetic.
   */
  setTokens: (tokens: TokenPair) => void;

  /**
   * Update ONLY the access token + expiry without touching the refresh token.
   * Used by the refresh flow when a retry of an in-flight request needs the
   * new access token synchronously from the store but we've already updated
   * the pair through setTokens. Kept for clarity and future use; currently
   * refresh always rotates both tokens so setTokens() is used.
   */
  setAccessToken: (accessToken: string, expiresInSeconds: number) => void;

  /** Wipe all auth state (logout or unrecoverable refresh failure). */
  clearTokens: () => void;
};

function readStoredExpiresAt(): number | null {
  const raw = sessionStorage.getItem(EXPIRES_AT_KEY);
  if (!raw) return null;
  const n = Number(raw);
  return Number.isFinite(n) ? n : null;
}

function getInitialAuthState() {
  return {
    accessToken: localStorage.getItem(ACCESS_TOKEN_KEY),
    refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY),
    expiresAt: readStoredExpiresAt(),
  };
}

// This store keeps auth tokens for the early frontend build.
// Later, refresh tokens should move to httpOnly cookies for stronger security.
//
// Token refresh (REST + WS) is orchestrated from @/api/authTokenManager so
// that axios interceptors, router guards and the WS layer all share the same
// in-flight lock and state transitions.
export const useAuthStore = create<AuthState>((set) => ({
  ...getInitialAuthState(),

  setTokens: (tokens) => {
    const expiresAt = Date.now() + tokens.expires_in * 1000;
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
    sessionStorage.setItem(EXPIRES_AT_KEY, String(expiresAt));
    set({
      accessToken: tokens.access_token,
      refreshToken: tokens.refresh_token,
      expiresAt,
    });
  },

  setAccessToken: (accessToken, expiresInSeconds) => {
    const expiresAt = Date.now() + expiresInSeconds * 1000;
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    sessionStorage.setItem(EXPIRES_AT_KEY, String(expiresAt));
    set({ accessToken, expiresAt });
  },

  clearTokens: () => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    sessionStorage.removeItem(EXPIRES_AT_KEY);
    set({ accessToken: null, refreshToken: null, expiresAt: null });
  },
}));

// Test helper. Keeping it exported avoids fragile test-only store hacks.
export function resetAuthStore() {
  useAuthStore.setState({ accessToken: null, refreshToken: null, expiresAt: null });
}
