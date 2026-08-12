import { apiRequest } from './http';

export type TokenPair = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
};

/**
 * Normalize the login/refresh response from the backend.
 *
 * The Go domain.TokenPair struct now has `json:"..."` tags that serialize
 * fields as snake_case (access_token/refresh_token/expires_in). For backward
 * compatibility with previous builds and any local backend that hasn't been
 * restarted with the updated struct, we also accept PascalCase.
 */
function normalizeTokens(raw: unknown): TokenPair {
  const r = raw as Record<string, unknown>;
  const access_token = (r.access_token ?? r.AccessToken) as string | undefined;
  const refresh_token = (r.refresh_token ?? r.RefreshToken) as string | undefined;
  const expires_in = (r.expires_in ?? r.ExpiresIn ?? 900) as number;
  if (!access_token || !refresh_token) {
    throw new Error(
      'Auth response missing tokens. Got keys: ' + (raw ? Object.keys(raw as object).join(',') : 'null'),
    );
  }
  return { access_token, refresh_token, expires_in };
}

export async function login(username: string, password: string): Promise<TokenPair> {
  // Public endpoint — skip Authorization header.
  const raw = await apiRequest<Record<string, unknown>>('/api/v1/auth/login', {
    method: 'POST',
    data: { username, password },
    accessToken: null,
  });
  return normalizeTokens(raw);
}

export async function register(username: string, password: string, confirmPassword: string) {
  // Public endpoint — skip Authorization header.
  return apiRequest('/api/v1/auth/register', {
    method: 'POST',
    data: { username, password, confirm_password: confirmPassword },
    accessToken: null,
  });
}

export async function refresh(refreshToken: string): Promise<TokenPair> {
  // The refresh endpoint takes the refresh token in the JSON body and does
  // NOT read an Authorization header — if we let the interceptor attach our
  // now-expired access token, the backend's middleware chain could reject
  // the request before it reaches the refresh controller. Opt out.
  const raw = await apiRequest<Record<string, unknown>>('/api/v1/auth/refresh', {
    method: 'POST',
    data: { refresh_token: refreshToken },
    accessToken: null,
  });
  return normalizeTokens(raw);
}

export async function logout() {
  // The Authorization header is attached automatically by the interceptor.
  // We DO NOT pass accessToken manually anymore.
  return apiRequest('/api/v1/auth/logout', {
    method: 'POST',
  });
}
