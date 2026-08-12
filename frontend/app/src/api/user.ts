import { apiRequest } from './http';

export type UserProfile = {
  user: {
    id: string;
    username: string;
    is_active: boolean;
    status: string;
    is_superuser: boolean;
    ban_count: number;
    banned_at: string | null;
    created_at: string;
    updated_at: string;
  };
  profile: {
    user_id: string;
    avatar_url: string;
    email: string;
    phone: string;
    email_verified: boolean;
    phone_verified: boolean;
    locale: string;
    background_sound: boolean;
    tool_sound: boolean;
    word_score: number;
    reputation_score: number;
    games_played: number;
    mvps: number;
    rank: string;
    created_at: string;
    updated_at: string;
  };
};

export type UpdateProfileInput = {
  avatar_url?: string;
  email?: string;
  phone?: string;
  locale?: string;
  background_sound?: boolean;
  tool_sound?: boolean;
};
// Theme is intentionally absent — it is owned by the frontend via localStorage
// (see stores/themeStore.ts) and is never sent to the backend.

export type ChangeUsernameInput = {
  username: string;
};

export type VerifyType = 'email' | 'phone';

const PREFIX = '/api/v1';

export function getProfile(): Promise<UserProfile> {
  return apiRequest<
    UserProfile | (Record<string, unknown> & { User?: UserProfile['user']; Profile?: UserProfile['profile'] })
  >(`${PREFIX}/user/profile`).then(normalizeUserProfile);
}

// Accept both `{user, profile}` (current) and `{User, Profile}` (legacy Go
// serialization when struct tags are missing) so we never crash the dashboard
// if the backend ever returns capitalized field names again.
function normalizeUserProfile(
  raw:
    | UserProfile
    | (Record<string, unknown> & { User?: UserProfile['user']; Profile?: UserProfile['profile'] }),
): UserProfile {
  const user = (raw as UserProfile).user ?? (raw as { User?: UserProfile['user'] }).User;
  const profile = (raw as UserProfile).profile ?? (raw as { Profile?: UserProfile['profile'] }).Profile;
  if (!user || !profile) {
    throw new Error('profile response is missing user or profile');
  }
  return { user, profile };
}

export function updateProfile(input: UpdateProfileInput): Promise<UserProfile['profile']> {
  return apiRequest<UserProfile['profile'], UpdateProfileInput>(`${PREFIX}/user/profile`, {
    method: 'PATCH',
    data: input,
  });
}

export function changeUsername(input: ChangeUsernameInput): Promise<{ message: string }> {
  return apiRequest<{ message: string }, ChangeUsernameInput>(`${PREFIX}/user/profile/username`, {
    method: 'POST',
    data: input,
  });
}

export function requestVerification(type: VerifyType): Promise<{ message: string }> {
  return apiRequest<{ message: string }, { type: VerifyType }>(`${PREFIX}/user/profile/verify/request`, {
    method: 'POST',
    data: { type },
  });
}

export function confirmVerification(type: VerifyType, code: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }, { type: VerifyType; code: string }>(
    `${PREFIX}/user/profile/verify/confirm`,
    { method: 'POST', data: { type, code } },
  );
}
