/**
 * authTokenManager
 *
 * Centralized orchestration for access/refresh token lifecycle. All code that
 * cares about token freshness (axios interceptor, router guards, WebSocket
 * layer) goes through this module so there is a single source of truth for
 * "is the token fresh?" and "is a refresh in flight?".
 *
 * Strategy:
 *  - REST API calls: REACTIVE. We attach the current access token to every
 *    request. On HTTP 401 we attempt a single refresh (with a promise lock so
 *    concurrent 401s collapse behind one /auth/refresh call), update the
 *    store, and replay the failed request(s).
 *  - WebSocket (in-game): PROACTIVE, but only while a WS is actually open.
 *    Call wsAcquireToken() before opening a socket; it refreshes if near
 *    expiry and starts a timer that refreshes ~1 minute before the new token
 *    expires. Call wsReleaseToken() when the socket closes and the timer
 *    stops. This keeps the session alive across multi-round games without
 *    endlessly sliding the session while the user is idle on a menu.
 *  - Router guards (hard refresh / deep link): ONE-SHOT check via
 *    getValidAccessToken({ silent: true }), which does NOT invoke the global
 *    failure callback. The router guard is responsible for issuing its own
 *    redirect() on failure.
 */
import { refresh as apiRefresh } from '@/api/auth';
import { useAuthStore } from '@/stores/authStore';

/**
 * How close to expiry (ms) we consider a token "stale" and proactively
 * refresh it. 60 seconds gives plenty of headroom for WS round-trips and
 * network jitter.
 */
const PROACTIVE_REFRESH_SKEW_MS = 60_000;

/**
 * How close to expiry (ms) we still bother refreshing at all. If the token
 * expires this soon or sooner, treat it as already expired (skip the timer,
 * refresh immediately).
 */
const EXPIRY_TOLERANCE_MS = 5_000;

// In-flight refresh promise lock. Null when nothing is happening. Non-null
// means some caller (interceptor, ws manager, router guard) is already
// talking to /auth/refresh and everyone else should await the same promise.
let refreshPromise: Promise<boolean> | null = null;

// Proactive refresh timer (for WebSocket sessions). Null when idle.
let proactiveTimer: ReturnType<typeof setTimeout> | null = null;

// Count of active WS "sessions" so that multiple sockets share the timer
// and we only stop it when the last one releases.
let wsRefCount = 0;

// Global callback for unrecoverable auth failures. The router interceptor
// and the proactive WS timer use this to bounce the user to /login when a
// refresh fails mid-session (outside of a router beforeLoad). Router guards
// pass silent:true and handle redirect themselves to avoid double firing.
let authFailureHandler: ((reason: AuthFailureReason) => void) | null = null;

export type AuthFailureReason = 'refresh_failed' | 'missing_refresh_token';

export function onAuthFailure(handler: (reason: AuthFailureReason) => void) {
  authFailureHandler = handler;
}

/** Test-only: reset module state between tests. */
export function __resetAuthTokenManager() {
  refreshPromise = null;
  if (proactiveTimer) {
    clearTimeout(proactiveTimer);
    proactiveTimer = null;
  }
  wsRefCount = 0;
  authFailureHandler = null;
}

/** Read the current access token synchronously, without attempting refresh. */
export function readAccessToken(): string | null {
  return useAuthStore.getState().accessToken;
}

/** Read the current refresh token synchronously. */
export function readRefreshToken(): string | null {
  return useAuthStore.getState().refreshToken;
}

function isTokenExpiredOrNearExpiry(expiresAt: number | null, skewMs = EXPIRY_TOLERANCE_MS): boolean {
  if (expiresAt == null) {
    // If we don't know when it expires, be pessimistic — if we have an access
    // token but no timestamp (e.g. stale localStorage from a previous app
    // version) treat as needing refresh.
    return Boolean(useAuthStore.getState().accessToken);
  }
  return Date.now() >= expiresAt - skewMs;
}

/**
 * Perform a single refresh against the backend and update the store.
 * @param silent - when true, do NOT invoke the global failure handler on
 *   error. Callers that already handle navigation (e.g. router guards) pass
 *   true to avoid double-redirect.
 * @returns true on success, false on failure (state has been cleared).
 */
async function performRefresh(silent: boolean): Promise<boolean> {
  const { refreshToken } = useAuthStore.getState();
  if (!refreshToken) {
    useAuthStore.getState().clearTokens();
    if (!silent) authFailureHandler?.('missing_refresh_token');
    return false;
  }

  try {
    const tokens = await apiRefresh(refreshToken);
    useAuthStore.getState().setTokens(tokens);
    return true;
  } catch {
    // Refresh failed — unrecoverable (expired/revoked/reuse detected).
    useAuthStore.getState().clearTokens();
    stopProactiveTimer();
    if (!silent) authFailureHandler?.('refresh_failed');
    return false;
  }
}

/**
 * Get a valid access token, refreshing if necessary. Uses the promise lock
 * so concurrent callers all await the same refresh request.
 *
 * @param opts.silent - when true, do NOT invoke the global failure handler
 *   on refresh failure (caller will handle it).
 * @returns A valid access token string, or null if refresh failed (state
 *   is cleared).
 */
export async function getValidAccessToken(opts: { silent?: boolean } = {}): Promise<string | null> {
  const { silent = false } = opts;
  const { accessToken, expiresAt } = useAuthStore.getState();
  if (accessToken && !isTokenExpiredOrNearExpiry(expiresAt)) {
    return accessToken;
  }

  if (refreshPromise) {
    const ok = await refreshPromise;
    return ok ? useAuthStore.getState().accessToken : null;
  }

  refreshPromise = performRefresh(silent).finally(() => {
    refreshPromise = null;
  });

  const ok = await refreshPromise;

  if (ok && wsRefCount > 0) {
    scheduleProactiveRefresh();
  }

  return ok ? useAuthStore.getState().accessToken : null;
}

/**
 * Variant used by axios response interceptor for retrying 401'd requests.
 * Always notifies the global failure handler on error (this is a mid-session
 * failure, not a navigation). Returns the new access token on success;
 * throws on failure.
 */
export async function refreshForRetry(): Promise<string> {
  const ok = refreshPromise
    ? await refreshPromise
    : await (refreshPromise = performRefresh(false).finally(() => {
        refreshPromise = null;
      }));

  if (ok && wsRefCount > 0) {
    scheduleProactiveRefresh();
  }

  if (!ok) throw new Error('auth: refresh failed');
  const token = useAuthStore.getState().accessToken;
  if (!token) throw new Error('auth: refresh failed');
  return token;
}

// --- Proactive refresh for WebSocket sessions -------------------------------

function stopProactiveTimer() {
  if (proactiveTimer) {
    clearTimeout(proactiveTimer);
    proactiveTimer = null;
  }
}

function scheduleProactiveRefresh() {
  stopProactiveTimer();
  const { expiresAt, accessToken } = useAuthStore.getState();
  if (!accessToken || expiresAt == null) return;

  const msUntilRefresh = expiresAt - Date.now() - PROACTIVE_REFRESH_SKEW_MS;
  // If we're already within the skew window, fire immediately (but still
  // asynchronously so the caller doesn't block on the network round-trip).
  const delay = Math.max(msUntilRefresh, 0);

  proactiveTimer = setTimeout(() => {
    proactiveTimer = null;
    // Directly perform the refresh rather than routing through
    // getValidAccessToken(): that function uses a small "expiry tolerance"
    // (5s) that would no-op when we intentionally fire 60s early. If a
    // refresh is already in flight (e.g. triggered by a concurrent 401),
    // we piggy-back on it via refreshPromise.
    void (async () => {
      const ok = refreshPromise
        ? await refreshPromise
        : await (refreshPromise = performRefresh(false).finally(() => {
            refreshPromise = null;
          }));
      if (ok) scheduleProactiveRefresh();
    })();
  }, delay);
}

/**
 * Acquire a valid access token for opening a WebSocket, and start the
 * proactive refresh timer so the token stays fresh for the duration of the
 * connection. Multiple concurrent acquisitions share one timer (ref-counted).
 */
export async function wsAcquireToken(): Promise<string | null> {
  wsRefCount += 1;
  const token = await getValidAccessToken({ silent: false });
  if (!token) {
    wsRefCount = Math.max(0, wsRefCount - 1);
    if (wsRefCount === 0) stopProactiveTimer();
    return null;
  }
  scheduleProactiveRefresh();
  return token;
}

/** Release one WebSocket session; stops the timer when the last one closes. */
export function wsReleaseToken() {
  wsRefCount = Math.max(0, wsRefCount - 1);
  if (wsRefCount === 0) {
    stopProactiveTimer();
  }
}
