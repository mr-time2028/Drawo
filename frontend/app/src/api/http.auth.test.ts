import MockAdapter from 'axios-mock-adapter';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { apiRequest, httpClient } from './http';
import { __resetAuthTokenManager, onAuthFailure } from './authTokenManager';
import { resetAuthStore, useAuthStore } from '@/stores/authStore';

const mock = new MockAdapter(httpClient);

afterEach(() => {
  mock.reset();
  localStorage.clear();
  sessionStorage.clear();
  resetAuthStore();
  __resetAuthTokenManager();
  vi.useRealTimers();
});

describe('httpClient auth interceptor', () => {
  it('auto-attaches the Bearer token from the store', async () => {
    useAuthStore.getState().setTokens({
      access_token: 'abc',
      refresh_token: 'r',
      expires_in: 900,
    });

    mock.onGet('/whoami').reply((config) => {
      expect(config.headers?.Authorization).toBe('Bearer abc');
      return [200, { user: 'hamid' }];
    });

    const result = await apiRequest<{ user: string }>('/whoami');
    expect(result).toEqual({ user: 'hamid' });
  });

  it('refreshes and retries once on 401', async () => {
    vi.useFakeTimers();
    useAuthStore.getState().setTokens({
      access_token: 'expired-access',
      refresh_token: 'good-refresh',
      // Make the token appear fresh so getValidAccessToken doesn't pre-refresh
      // — we're testing the response interceptor path.
      expires_in: 900,
    });

    let callCount = 0;
    mock.onGet('/me').reply((config) => {
      callCount += 1;
      if (callCount === 1) {
        // First call with expired token returns 401.
        expect(config.headers?.Authorization).toBe('Bearer expired-access');
        return [401, { message: 'unauthorized' }];
      }
      // Second call should carry the new token.
      expect(config.headers?.Authorization).toBe('Bearer new-access');
      return [200, { ok: true }];
    });

    mock.onPost('/api/v1/auth/refresh').reply(200, {
      access_token: 'new-access',
      refresh_token: 'new-refresh',
      expires_in: 900,
    });

    const result = await apiRequest<{ ok: boolean }>('/me');
    expect(result).toEqual({ ok: true });
    expect(callCount).toBe(2);
    expect(useAuthStore.getState().accessToken).toBe('new-access');
  });

  it('does NOT loop infinitely — gives up after one retry', async () => {
    vi.useFakeTimers();
    useAuthStore.getState().setTokens({
      access_token: 'access',
      refresh_token: 'refresh',
      expires_in: 900,
    });

    // /me always returns 401, even with the new token.
    mock.onGet('/me').reply(401, { message: 'nope' });
    mock.onPost('/api/v1/auth/refresh').reply(200, {
      access_token: 'new-access',
      refresh_token: 'new-refresh',
      expires_in: 900,
    });

    const failureHandler = vi.fn();
    onAuthFailure(failureHandler);

    await expect(apiRequest('/me')).rejects.toMatchObject({ status: 401 });
    // one original + one retry = 2, NOT more.
    const meCalls = mock.history.get.filter((r) => r.url === '/me');
    expect(meCalls).toHaveLength(2);
    // refresh succeeded but even after retry we got 401 — that is NOT an
    // auth-refresh failure, so the failure handler shouldn't fire.
    expect(failureHandler).not.toHaveBeenCalled();
  });

  it('does NOT try to refresh when the original request was a public endpoint', async () => {
    // No token in the store; we call a public endpoint that returns 401 for
    // a non-auth reason (e.g. wrong credentials). Refresh must NOT be called.
    mock.onPost('/api/v1/auth/login').reply(401, { message: 'invalid credentials' });

    const refreshSpy = vi.fn();
    mock.onPost('/api/v1/auth/refresh').reply(() => {
      refreshSpy();
      return [500, {}];
    });

    await expect(
      apiRequest('/api/v1/auth/login', {
        method: 'POST',
        accessToken: null, // opt out as our auth endpoints do
        data: { username: 'x', password: 'y' },
      }),
    ).rejects.toMatchObject({ status: 401 });

    expect(refreshSpy).not.toHaveBeenCalled();
  });
});
