import MockAdapter from 'axios-mock-adapter';
import { afterEach, describe, expect, it } from 'vitest';

import {
  buildInviteURL,
  closeRoom,
  createRoom,
  getRoom,
  getRoomByCode,
  joinRoomByCode,
  leaveRoom,
  parseWordList,
  startRoom,
} from './room';
import { httpClient } from './http';

const mock = new MockAdapter(httpClient, { onNoMatch: 'passthrough' });

afterEach(() => {
  mock.reset();
});

const sampleRoom = {
  id: 'r-1',
  name: 'Test',
  invite_code: 'ABCDEF',
  owner_id: 'u-1',
  type: 'private' as const,
  has_password: false,
  language: 'en',
  word_source: 'default' as const,
  state: 'lobby' as const,
  min_players: 2,
  max_players: 8,
  round_time: 80,
  max_rounds: 3,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('room API client', () => {
  it('createRoom POSTs to /rooms and defaults room_type to private', async () => {
    mock.onPost('/api/v1/rooms').reply((config) => {
      const body = JSON.parse(config.data as string);
      expect(body.room_type).toBe('private');
      expect(body.name).toBe('My Room');
      expect(body.max_players).toBe(6);
      return [201, sampleRoom];
    });
    const r = await createRoom({
      name: 'My Room',
      max_players: 6,
      max_rounds: 3,
      round_time: 80,
      word_source: 'default',
    });
    expect(r.id).toBe('r-1');
  });

  it('getRoom GETs /rooms/:id with URL-safe id', async () => {
    mock.onGet('/api/v1/rooms/r%2F1').reply(200, sampleRoom);
    const r = await getRoom('r/1');
    expect(r.name).toBe('Test');
  });

  it('getRoomByCode GETs /rooms/by-code/:code', async () => {
    mock.onGet('/api/v1/rooms/by-code/ABCDEF').reply(200, sampleRoom);
    const r = await getRoomByCode('ABCDEF');
    expect(r.invite_code).toBe('ABCDEF');
  });

  it('joinRoomByCode POSTs password when provided', async () => {
    mock.onPost('/api/v1/rooms/by-code/ABCDEF/join').reply((config) => {
      expect(JSON.parse(config.data as string)).toEqual({ password: 'pw' });
      return [200, sampleRoom];
    });
    const r = await joinRoomByCode('ABCDEF', { password: 'pw' });
    expect(r.id).toBe('r-1');
  });

  it('joinRoomByCode sends undefined password as JSON null/undefined', async () => {
    mock.onPost('/api/v1/rooms/by-code/ABCDEF/join').reply((config) => {
      const body = JSON.parse(config.data as string);
      expect(body).toEqual({ password: undefined });
      return [200, sampleRoom];
    });
    await joinRoomByCode('ABCDEF');
  });

  it('startRoom POSTs empty object to /rooms/:id/start', async () => {
    mock.onPost('/api/v1/rooms/r-1/start').reply(200, { ...sampleRoom, state: 'playing' });
    const r = await startRoom('r-1');
    expect(r.state).toBe('playing');
  });

  it('leaveRoom POSTs to /rooms/:id/leave', async () => {
    mock.onPost('/api/v1/rooms/r-1/leave').reply(200, sampleRoom);
    const r = await leaveRoom('r-1');
    expect(r.id).toBe('r-1');
  });

  it('closeRoom POSTs to /rooms/:id/close', async () => {
    mock.onPost('/api/v1/rooms/r-1/close').reply(200, { message: 'room closed' });
    const r = await closeRoom('r-1');
    expect(r.message).toBe('room closed');
  });

  it('buildInviteURL uses window.location.origin', () => {
    expect(buildInviteURL('XYZ789')).toBe(`${window.location.origin}/r/XYZ789`);
  });

  it('parseWordList splits on commas, Persian commas and newlines', () => {
    expect(parseWordList('cat,dog،mouse\nfish')).toEqual(['cat', 'dog', 'mouse', 'fish']);
  });
  it('parseWordList trims whitespace and drops empties', () => {
    expect(parseWordList('  a ,  b\n، ,c  ')).toEqual(['a', 'b', 'c']);
  });
  it('parseWordList returns empty array for empty/undefined input', () => {
    expect(parseWordList('')).toEqual([]);
  });
});
