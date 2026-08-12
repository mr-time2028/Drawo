import { apiRequest } from './http';
import { clearGuestSession, saveGuestSession, type GuestSession } from './guestTokenManager';

export type RoomWordSource = 'default' | 'category' | 'custom';

export type CustomCategory = {
  name: string;
  words: Record<string | number, string[]>; // keys "1"|"2"|"3" or 1|2|3
};

export type CreateRoomInput = {
  name: string;
  password?: string;
  language?: string;
  min_players?: number;
  max_players: number;
  max_rounds: number;
  round_time: number;
  word_source: RoomWordSource;
  room_type?: 'private' | 'public';
  custom_categories?: CustomCategory[];
};

export type Room = {
  id: string;
  name: string;
  invite_code: string;
  owner_id: string;
  type: 'public' | 'private';
  has_password: boolean;
  language: string;
  word_source: RoomWordSource;
  state: 'lobby' | 'playing' | 'finished' | 'closed';
  min_players: number;
  max_players: number;
  round_time: number;
  max_rounds: number;
  current_round?: number;
  custom_categories?: CustomCategory[];
  custom_word_count?: number;
  player_count?: number;
  created_at: string;
  updated_at: string;
};

const PREFIX = '/api/v1';

export function createRoom(input: CreateRoomInput): Promise<Room> {
  return apiRequest<Room, CreateRoomInput>(`${PREFIX}/rooms`, {
    method: 'POST',
    data: { ...input, room_type: input.room_type ?? 'private' },
  });
}

export function getRoom(id: string): Promise<Room> {
  return apiRequest<Room>(`${PREFIX}/rooms/${encodeURIComponent(id)}`);
}

export function getRoomByCode(code: string): Promise<Room> {
  // Public invite lookup — DO NOT send any Authorization header. If a stale
  // access_token is sitting in localStorage from a previous session the
  // backend may 401 it, and the axios response interceptor would then try
  // refresh, fail, and fire the global auth-failure handler — which
  // navigates the user away from the invite page even though invite pages
  // are supposed to work for anonymous guests. Skip auth entirely for this
  // public endpoint.
  return apiRequest<Room>(`${PREFIX}/rooms/by-code/${encodeURIComponent(code)}`, {
    accessToken: null,
  });
}

export type JoinByCodeResult = Room & {
  is_guest?: boolean;
  guest_token?: string;
  guest_id?: string;
  nickname?: string;
};

export type JoinByCodeInput = {
  password?: string;
  nickname?: string;
};

/**
 * Join a private room by invite code. For anonymous users (no access token),
 * pass a `nickname` — the backend will issue a short-lived, room-bound
 * guest_token which this helper persists via guestTokenManager.
 */
export async function joinRoomByCode(code: string, opts: JoinByCodeInput = {}): Promise<JoinByCodeResult> {
  // Clear any stale guest session before attempting a new join; if this join
  // succeeds as a guest we'll overwrite it below.
  clearGuestSession();

  const res = await apiRequest<JoinByCodeResult, JoinByCodeInput>(
    `${PREFIX}/rooms/by-code/${encodeURIComponent(code)}/join`,
    {
      method: 'POST',
      data: {
        password: opts.password,
        nickname: opts.nickname,
      },
      // Anonymous guest join MUST NOT attach a stale/expired Bearer token.
      // When the caller has no logged-in session, we explicitly skip auth so
      // OptionalAuth on the backend treats the request as anonymous.
      accessToken: opts.nickname ? null : undefined,
    },
  );

  if (res.is_guest && res.guest_token && res.guest_id && res.nickname) {
    const session: GuestSession = {
      guestToken: res.guest_token,
      guestID: res.guest_id,
      nickname: res.nickname,
      roomID: res.id,
    };
    saveGuestSession(session);
  }

  return res;
}

export function startRoom(id: string): Promise<Room> {
  return apiRequest<Room, Record<string, never>>(`${PREFIX}/rooms/${encodeURIComponent(id)}/start`, {
    method: 'POST',
    data: {},
  });
}

export function leaveRoom(id: string): Promise<Room> {
  return apiRequest<Room, Record<string, never>>(`${PREFIX}/rooms/${encodeURIComponent(id)}/leave`, {
    method: 'POST',
    data: {},
  });
}

export function closeRoom(id: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }, Record<string, never>>(
    `${PREFIX}/rooms/${encodeURIComponent(id)}/close`,
    {
      method: 'POST',
      data: {},
    },
  );
}

export function buildInviteURL(code: string): string {
  return `${window.location.origin}/r/${code}`;
}

/**
 * Splits a user-pasted word list (comma, Persian comma, or newline separated)
 * into an array of trimmed, non-empty words. Exported for tests.
 */
export function parseWordList(input: string): string[] {
  if (!input) return [];
  return input
    .split(/[\n,،]/u)
    .map((w) => w.trim())
    .filter((w) => w.length > 0);
}
