/**
 * Tests for useRoomSocket React hook.
 *
 * We stub connectGameSocket so we can drive a fake socket from the test,
 * avoiding any real WebSocket / timer concerns.
 */
import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { connectGameSocket, type GameSocket, type ServerEnvelope } from './gameSocket';
import { useRoomSocket } from './useRoomSocket';
import * as guestMod from './guestTokenManager';
import * as authMod from './authTokenManager';

vi.mock('./gameSocket', () => ({
  connectGameSocket: vi.fn(),
}));

type FakeCtrlSocket = GameSocket & {
  _messages: Array<(env: ServerEnvelope) => void>;
  _statusCbs: Array<(status: string, detail?: string) => void>;
  _closed: boolean;
  _sent: Array<{ type: string; payload?: unknown }>;
  // Deliver a server envelope to the hook's onMessage.
  deliver: (env: ServerEnvelope) => void;
  setStatusOverride: (cb: (s: string, d?: string) => void) => void;
};

function makeFakeSocket(): FakeCtrlSocket {
  let onMessage: ((env: ServerEnvelope) => void) | null = null;
  let onStatus: ((s: string, d?: string) => void) | null = null;
  const sock: FakeCtrlSocket = {
    _messages: [],
    _statusCbs: [],
    _closed: false,
    _sent: [],
    close: () => {
      sock._closed = true;
    },
    send: (type: string, payload?: unknown) => {
      sock._sent.push({ type, payload });
    },
    getStatus: () => (sock._closed ? 'closed' : 'open'),
    deliver: (env: ServerEnvelope) => {
      if (onMessage) onMessage(env);
    },
    setStatusOverride: (cb) => {
      onStatus = cb;
    },
  };
  // Capture the callbacks by spying on the connect promise resolution.
  (connectGameSocket as unknown as ReturnType<typeof vi.fn>).mockImplementation(
    (opts: { onMessage?: (e: ServerEnvelope) => void; onStatusChange?: (s: string, d?: string) => void }) => {
      onMessage = opts.onMessage ?? null;
      onStatus = opts.onStatusChange ?? null;
      return Promise.resolve(sock);
    },
  );
  return sock;
}

describe('useRoomSocket', () => {
  beforeEach(() => {
    vi.mocked(connectGameSocket).mockReset();
    vi.spyOn(guestMod, 'readGuestSession').mockReturnValue(null);
    vi.spyOn(authMod, 'readAccessToken').mockReturnValue('user-access');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns idle state when roomId is undefined', () => {
    const { result } = renderHook(() => useRoomSocket(undefined));
    expect(result.current.status).toBe('idle');
    expect(result.current.players).toEqual([]);
    expect(connectGameSocket).not.toHaveBeenCalled();
  });

  it('errors when no access token and no guest session', () => {
    vi.mocked(authMod.readAccessToken).mockReturnValue(null);
    vi.mocked(guestMod.readGuestSession).mockReturnValue(null);
    const { result } = renderHook(() => useRoomSocket('r1'));
    expect(result.current.status).toBe('error');
    expect(result.current.error).toBe('not authenticated');
    expect(connectGameSocket).not.toHaveBeenCalled();
  });

  it('transitions connecting → open, normalizes players from game_state, and exposes send()', async () => {
    const sock = makeFakeSocket();
    const { result } = renderHook(() => useRoomSocket('r1', 'INV'));

    // Right after mount, status is connecting.
    await waitFor(() => expect(result.current.status).not.toBe('idle'));
    expect(result.current.status).toBe('connecting');
    expect(connectGameSocket).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'private', roomID: 'r1', inviteCode: 'INV', guest: null }),
    );

    // Simulate server sending joined (which updates status to open).
    act(() => {
      sock.deliver({ type: 'joined' });
    });
    expect(result.current.status).toBe('open');

    // Game state populates players with is_owner/is_guest flags.
    act(() => {
      sock.deliver({
        type: 'game_state',
        payload: {
          state: 'waiting_for_players',
          players: [
            { user_id: 'u1', username: 'Owner', score: 0, is_drawer: false, is_online: true, is_owner: true },
            {
              user_id: 'guest:g1',
              username: 'Alice',
              score: 0,
              is_drawer: false,
              is_online: true,
              is_guest: true,
            },
          ],
          min_players: 3,
          max_players: 10,
        },
      });
    });
    expect(result.current.gameState).toBe('waiting_for_players');
    expect(result.current.minPlayers).toBe(3);
    expect(result.current.maxPlayers).toBe(10);
    expect(result.current.players).toHaveLength(2);
    expect(result.current.players[0]!.is_owner).toBe(true);
    expect(result.current.players[1]!.is_guest).toBe(true);
    expect(result.current.players[1]!.username).toBe('Alice');

    // Send forwards to socket.
    act(() => {
      result.current.send('game', { event: 'start' });
    });
    expect(sock._sent).toContainEqual({ type: 'game', payload: { event: 'start' } });
  });

  it('uses guest session when present and room matches', async () => {
    vi.mocked(authMod.readAccessToken).mockReturnValue(null);
    vi.mocked(guestMod.readGuestSession).mockReturnValue({
      guestToken: 'gt',
      guestID: 'guest:g',
      nickname: 'Bob',
      roomID: 'r2',
    });
    makeFakeSocket();
    renderHook(() => useRoomSocket('r2'));
    await waitFor(() => expect(connectGameSocket).toHaveBeenCalled());
    expect(vi.mocked(connectGameSocket).mock.calls[0]![0]).toMatchObject({
      roomID: 'r2',
      guest: { guestToken: 'gt', guestID: 'guest:g', nickname: 'Bob', roomID: 'r2' },
    });
  });

  it('reports error on error envelope', async () => {
    const sock = makeFakeSocket();
    const { result } = renderHook(() => useRoomSocket('r1'));
    await waitFor(() => expect(connectGameSocket).toHaveBeenCalled());
    act(() => {
      sock.deliver({ type: 'error', payload: { message: 'kicked' } });
    });
    expect(result.current.error).toBe('kicked');
  });

  it('annotateOwner patches is_owner flag for given ownerID', async () => {
    const sock = makeFakeSocket();
    const { result } = renderHook(() => useRoomSocket('r1'));
    await waitFor(() => expect(connectGameSocket).toHaveBeenCalled());
    act(() => {
      sock.deliver({
        type: 'game_state',
        payload: {
          state: 'waiting_for_players',
          players: [
            { user_id: 'u1', score: 0, is_drawer: false, is_online: true },
            { user_id: 'u2', score: 0, is_drawer: false, is_online: true },
          ],
        },
      });
    });
    expect(result.current.players.find((p) => p.user_id === 'u1')!.is_owner).toBe(false);
    act(() => {
      result.current.annotateOwner('u1');
    });
    expect(result.current.players.find((p) => p.user_id === 'u1')!.is_owner).toBe(true);
    expect(result.current.players.find((p) => p.user_id === 'u2')!.is_owner).toBe(false);
  });

  it('cleans up (closes socket) on unmount', async () => {
    const sock = makeFakeSocket();
    const { result, unmount } = renderHook(() => useRoomSocket('r1'));
    await waitFor(() => expect(connectGameSocket).toHaveBeenCalled());
    act(() => {
      sock.deliver({ type: 'joined' });
    });
    expect(result.current.status).toBe('open');
    expect(sock._closed).toBe(false);
    unmount();
    expect(sock._closed).toBe(true);
  });

  it('surfaces connection errors from connectGameSocket rejection', async () => {
    vi.mocked(connectGameSocket).mockRejectedValue(new Error('network down'));
    const { result } = renderHook(() => useRoomSocket('r1'));
    await waitFor(() => expect(result.current.status).toBe('error'));
    expect(result.current.error).toBe('network down');
  });
});
