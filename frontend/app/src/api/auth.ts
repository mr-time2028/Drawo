import { apiRequest } from './http';

export type TokenPair = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
};

export async function login(username: string, password: string): Promise<TokenPair> {
  return apiRequest<TokenPair>('/api/v1/auth/login', {
    method: 'POST',
    data: { username, password },
  });
}

export async function register(username: string, password: string, confirmPassword: string) {
  return apiRequest('/api/v1/auth/register', {
    method: 'POST',
    data: { username, password, confirm_password: confirmPassword },
  });
}

export async function refresh(refreshToken: string): Promise<TokenPair> {
  return apiRequest<TokenPair>('/api/v1/auth/refresh', {
    method: 'POST',
    data: { refresh_token: refreshToken },
  });
}

export async function logout(accessToken: string) {
  return apiRequest('/api/v1/auth/logout', {
    method: 'POST',
    accessToken,
  });
}
