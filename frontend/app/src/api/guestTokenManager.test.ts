import { afterEach, describe, expect, it } from 'vitest';

import {
  clearGuestSession,
  readGuestSession,
  readGuestToken,
  saveGuestSession,
} from './guestTokenManager';

afterEach(() => {
  localStorage.clear();
});

describe('guestTokenManager', () => {
  it('persists and reads a full guest session', () => {
    expect(readGuestSession()).toBeNull();
    expect(readGuestToken()).toBeNull();
    saveGuestSession({
      guestToken: 'tok-1',
      guestID: 'guest:abc',
      nickname: 'Alice',
      roomID: 'room-1',
    });
    expect(readGuestToken()).toBe('tok-1');
    const s = readGuestSession();
    expect(s).toEqual({
      guestToken: 'tok-1',
      guestID: 'guest:abc',
      nickname: 'Alice',
      roomID: 'room-1',
    });
  });

  it('returns null when any field is missing', () => {
    saveGuestSession({
      guestToken: 'tok',
      guestID: 'gid',
      nickname: 'Bob',
      roomID: 'r',
    });
    localStorage.removeItem('drawo.guest_nickname');
    expect(readGuestSession()).toBeNull();
  });

  it('clearGuestSession wipes all keys', () => {
    saveGuestSession({
      guestToken: 'tok',
      guestID: 'gid',
      nickname: 'Bob',
      roomID: 'r',
    });
    clearGuestSession();
    expect(readGuestSession()).toBeNull();
    expect(readGuestToken()).toBeNull();
  });
});
