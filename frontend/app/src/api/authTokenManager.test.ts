import MockAdapter from 'axios-mock-adapter';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { httpClient } from '@/api/http';
import { resetAuthStore, useAuthStore } from '@/stores/authStore';

import {
  __resetAuthTokenManager,
  getValidAccessToken,
  onAuthFailure,
  readAccessToken,
  refreshForRetry,
  wsAcquireToken,
  wsReleaseToken,
} from './authTokenManager';

const mock = new MockAdapter(httpClient);

beforeEach(() => {
  vi.useFakeTimers();
  // Set a deterministic "now" so expiresAt arithmetic is predictable.
  vi.setSystemTime(0);
});

afterEach(() => {
  mock.reset();
  localStorage.clear();
  sessionStorage.clear();
  resetAuthStore();
  __resetAuthTokenManager();
  vi.useRealTimers();
});

describe('authTokenManager', () => {
  it('returns the existing access token when it is fresh', async () => {
    useAuthStore.getState().setTokens({
      access_token: 'fresh-access',
      refresh_token: 'refresh',
      expires_in: 900,
    });
    mock.onPost('/api/v1/auth/refresh').reply(() => {
      expect.unreachable('refresh should not be called for a fresh token');
      return [500, {}];
    });

    const token = await getValidAccessToken();
    expect(token).toBe('fresh-access');
  });

  it('refreshes when access token is expired (or nearly expired)', async () => {
    // expires_in is very short.
    useAuthStore.getState().setTokens({
      access_token: 'old-access',
      refresh_token: 'old-refresh',
      expires_in: 1, // 1 second
    });

    // Advance past expiry.
    vi.advanceTimersByTime(2000);

    mock.onPost('/api/v1/auth/refresh').reply(200, {
      access_token: 'new-access',
      refresh_token: 'new-refresh',
      expires_in: 900,
    });

    const token = await getValidAccessToken();
    expect(token).toBe('new-access');
    expect(readAccessToken()).toBe('new-access');
    expect(useAuthStore.getState().refreshToken).toBe('new-refresh');
  });

  it('collapses concurrent refresh requests into a single network call', async () => {
    useAuthStore.getState().setTokens({
      access_token: 'old-access',
      refresh_token: 'old-refresh',
      expires_in: 1,
    });
    vi.advanceTimersByTime(2000);

    let refreshCallCount = 0;
    mock.onPost('/api/v1/auth/refresh').reply(() => {
      refreshCallCount += 1;
      return [200, { access_token: 'new-access', refresh_token: 'new-refresh', expires_in: 900 }];
    });

    const [a, b, c] = await Promise.all([getValidAccessToken(), getValidAccessToken(), refreshForRetry()]);

    expect(refreshCallCount).toBe(1);
    expect(a).toBe('new-access');
    expect(b).toBe('new-access');
    expect(c).toBe('new-access');
  });

  it('clears auth state and fires failure handler when refresh fails', async () => {
    useAuthStore.getState().setTokens({
      access_token: 'old-access',
      refresh_token: 'bad-refresh',
      expires_in: 1,
    });
    vi.advanceTimersByTime(2000);

    mock.onPost('/api/v1/auth/refresh').reply(401, { message: 'invalid refresh token' });

    const failureHandler = vi.fn();
    onAuthFailure(failureHandler);

    const token = await getValidAccessToken();
    expect(token).toBeNull();
    expect(useAuthStore.getState().accessToken).toBeNull();
    expect(useAuthStore.getState().refreshToken).toBeNull();
    expect(failureHandler).toHaveBeenCalledWith('refresh_failed');
  });

  it('schedules proactive refresh while WS is held, cancels on release', async () => {
    // expires_in=120s → expires at t=120_000ms. Proactive skew is 60_000ms, so
    // the timer should fire at t=60_000ms.
    const EXPIRES_IN_SEC = 120;
    const SKEW_MS = 60_000;
    useAuthStore.getState().setTokens({
      access_token: 'access',
      refresh_token: 'refresh',
      expires_in: EXPIRES_IN_SEC,
    });

    mock.onPost('/api/v1/auth/refresh').reply(200, {
      access_token: 'access-2',
      refresh_token: 'refresh-2',
      expires_in: EXPIRES_IN_SEC,
    });

    const token = await wsAcquireToken();
    expect(token).toBe('access');

    // Before the scheduled time, no refresh should have happened.
    vi.advanceTimersByTime(EXPIRES_IN_SEC * 1000 - SKEW_MS - 1);
    // Flush any microtasks without running more timers.
    await vi.advanceTimersByTimeAsync(0);
    expect(mock.history.post.filter((r) => r.url === '/api/v1/auth/refresh')).toHaveLength(0);

    // Advance exactly to the scheduled firing point and wait for the
    // (async) refresh to complete. We do NOT use runAllTimersAsync because
    // after a successful refresh the timer re-arms itself (sliding window)
    // which, in fake-timer land, would loop forever.
    await vi.advanceTimersByTimeAsync(10);

    expect(mock.history.post.filter((r) => r.url === '/api/v1/auth/refresh')).toHaveLength(1);
    expect(useAuthStore.getState().accessToken).toBe('access-2');

    wsReleaseToken();
    // After release, stop the timer so the test doesn't leak; further
    // advances must NOT trigger refresh.
    mock.resetHistory();
    await vi.advanceTimersByTimeAsync(1_000_000);
    expect(mock.history.post.filter((r) => r.url === '/api/v1/auth/refresh')).toHaveLength(0);
  });
});
