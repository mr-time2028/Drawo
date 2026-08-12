import { afterEach, describe, expect, it } from 'vitest';

import { useAuthStore, resetAuthStore } from './authStore';

afterEach(() => {
  localStorage.clear();
  sessionStorage.clear();
  resetAuthStore();
});

describe('authStore', () => {
  it('stores a TokenPair and derives expiresAt', () => {
    const beforeSet = Date.now();
    useAuthStore.getState().setTokens({
      access_token: 'access',
      refresh_token: 'refresh',
      expires_in: 900,
    });
    const afterSet = Date.now();

    expect(useAuthStore.getState().accessToken).toBe('access');
    expect(useAuthStore.getState().refreshToken).toBe('refresh');
    expect(localStorage.getItem('drawo.access_token')).toBe('access');
    expect(localStorage.getItem('drawo.refresh_token')).toBe('refresh');

    // expiresAt should be ~15 minutes from now.
    const expiresAt = useAuthStore.getState().expiresAt;
    expect(expiresAt).toBeTypeOf('number');
    expect(expiresAt!).toBeGreaterThanOrEqual(beforeSet + 899_000);
    expect(expiresAt!).toBeLessThanOrEqual(afterSet + 901_000);
    expect(Number(sessionStorage.getItem('drawo.expires_at'))).toBe(expiresAt);
  });

  it('clears tokens, refresh token, and expiresAt', () => {
    useAuthStore.getState().setTokens({
      access_token: 'access',
      refresh_token: 'refresh',
      expires_in: 900,
    });

    useAuthStore.getState().clearTokens();

    expect(useAuthStore.getState().accessToken).toBeNull();
    expect(useAuthStore.getState().refreshToken).toBeNull();
    expect(useAuthStore.getState().expiresAt).toBeNull();
    expect(localStorage.getItem('drawo.access_token')).toBeNull();
    expect(localStorage.getItem('drawo.refresh_token')).toBeNull();
    expect(sessionStorage.getItem('drawo.expires_at')).toBeNull();
  });

  it('setAccessToken updates only the access token + expiry', () => {
    useAuthStore.getState().setTokens({
      access_token: 'first-access',
      refresh_token: 'refresh',
      expires_in: 900,
    });

    const beforeSet = Date.now();
    useAuthStore.getState().setAccessToken('second-access', 60);
    const afterSet = Date.now();

    expect(useAuthStore.getState().accessToken).toBe('second-access');
    // refresh token stays intact
    expect(useAuthStore.getState().refreshToken).toBe('refresh');
    const expiresAt = useAuthStore.getState().expiresAt;
    expect(expiresAt).toBeGreaterThanOrEqual(beforeSet + 59_000);
    expect(expiresAt).toBeLessThanOrEqual(afterSet + 61_000);
  });
});
