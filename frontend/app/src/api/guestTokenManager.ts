// Guest tokens are short-lived, room-bound tokens for anonymous players
// joining a private room via invite link. Unlike access tokens they do NOT
// grant account access; they only let the bearer connect to the WS and play
// in the single room the token was issued for.

const GUEST_TOKEN_KEY = 'drawo.guest_token';
const GUEST_ID_KEY = 'drawo.guest_id';
const GUEST_NICKNAME_KEY = 'drawo.guest_nickname';
const GUEST_ROOM_KEY = 'drawo.guest_room_id';

export type GuestSession = {
  guestToken: string;
  guestID: string;
  nickname: string;
  roomID: string;
};

export function saveGuestSession(s: GuestSession) {
  localStorage.setItem(GUEST_TOKEN_KEY, s.guestToken);
  localStorage.setItem(GUEST_ID_KEY, s.guestID);
  localStorage.setItem(GUEST_NICKNAME_KEY, s.nickname);
  localStorage.setItem(GUEST_ROOM_KEY, s.roomID);
}

export function readGuestToken(): string | null {
  return localStorage.getItem(GUEST_TOKEN_KEY);
}

export function readGuestSession(): GuestSession | null {
  const t = localStorage.getItem(GUEST_TOKEN_KEY);
  const id = localStorage.getItem(GUEST_ID_KEY);
  const nick = localStorage.getItem(GUEST_NICKNAME_KEY);
  const rid = localStorage.getItem(GUEST_ROOM_KEY);
  if (!t || !id || !nick || !rid) return null;
  return { guestToken: t, guestID: id, nickname: nick, roomID: rid };
}

export function clearGuestSession() {
  localStorage.removeItem(GUEST_TOKEN_KEY);
  localStorage.removeItem(GUEST_ID_KEY);
  localStorage.removeItem(GUEST_NICKNAME_KEY);
  localStorage.removeItem(GUEST_ROOM_KEY);
}

export function __resetGuestTokenManager() {
  clearGuestSession();
}
