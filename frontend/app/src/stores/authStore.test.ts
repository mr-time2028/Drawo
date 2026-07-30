import { describe, expect, it } from 'vitest';

import { useAuthStore } from './authStore';

afterEach(() => {
  localStorage.clear();
  useAuthStore.getState().clearTokens();
});

describe('authStore', () => {
  it('stores and clears tokens', () => {
    useAuthStore.getState().setTokens('access', 'refresh');

    expect(useAuthStore.getState().accessToken).toBe('access');
    expect(useAuthStore.getState().refreshToken).toBe('refresh');
    expect(localStorage.getItem('drawo.access_token')).toBe('access');
    expect(localStorage.getItem('drawo.refresh_token')).toBe('refresh');

    useAuthStore.getState().clearTokens();

    expect(useAuthStore.getState().accessToken).toBeNull();
    expect(useAuthStore.getState().refreshToken).toBeNull();
    expect(localStorage.getItem('drawo.access_token')).toBeNull();
    expect(localStorage.getItem('drawo.refresh_token')).toBeNull();
  });
});
