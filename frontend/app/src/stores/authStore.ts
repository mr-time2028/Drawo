import { create } from 'zustand';

const ACCESS_TOKEN_KEY = 'drawo.access_token';
const REFRESH_TOKEN_KEY = 'drawo.refresh_token';

type AuthState = {
  accessToken: string | null;
  refreshToken: string | null;
  setTokens: (accessToken: string, refreshToken: string) => void;
  clearTokens: () => void;
};

function getInitialAuthState() {
  return {
    accessToken: localStorage.getItem(ACCESS_TOKEN_KEY),
    refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY),
  };
}

// This store keeps auth tokens for the early frontend build.
// Later, refresh tokens should move to httpOnly cookies for stronger security.
//
// NOTE: The backend /auth/logout call lives in the caller (currently
// DashboardPage.handleLogout) so that network + state-clearing can be ordered
// precisely: first POST to the backend while the token is still in state and
// the component is mounted, THEN clearTokens() and navigate.
export const useAuthStore = create<AuthState>((set) => ({
  ...getInitialAuthState(),

  setTokens: (accessToken, refreshToken) => {
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    set({ accessToken, refreshToken });
  },

  clearTokens: () => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    set({ accessToken: null, refreshToken: null });
  },
}));

// Test helper. Keeping it exported avoids fragile test-only store hacks.
export function resetAuthStore() {
  useAuthStore.setState({ accessToken: null, refreshToken: null });
}
