import { apiRequest } from './http';

export type TokenPair = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
};

/**
 * Normalize the login/refresh response from the backend.
 *
 * The Go domain.TokenPair struct currently lacks `json:"..."` tags, so it
 * serializes with PascalCase field names (AccessToken/RefreshToken/ExpiresIn).
 * We also accept the documented snake_case names for when the backend gets
 * the tags added. This defensive normalization keeps frontend code working
 * through either shape.
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
  const raw = await apiRequest<Record<string, unknown>>('/api/v1/auth/login', {
    method: 'POST',
    data: { username, password },
  });
  return normalizeTokens(raw);
}

export async function register(username: string, password: string, confirmPassword: string) {
  return apiRequest('/api/v1/auth/register', {
    method: 'POST',
    data: { username, password, confirm_password: confirmPassword },
  });
}

export async function refresh(refreshToken: string): Promise<TokenPair> {
  const raw = await apiRequest<Record<string, unknown>>('/api/v1/auth/refresh', {
    method: 'POST',
    data: { refresh_token: refreshToken },
  });
  return normalizeTokens(raw);
}

export async function logout(accessToken: string) {
  return apiRequest('/api/v1/auth/logout', {
    method: 'POST',
    accessToken,
  });
}
