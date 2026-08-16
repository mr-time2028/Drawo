/**
 * React hook that opens the Drawo WebSocket for a specific room and exposes
 * live connection state + player list for the lobby.
 *
 * Works for both registered users and guest sessions (uses guestTokenManager
 * to detect and pass the guest token).
 */
import { useCallback, useEffect, useRef, useState } from 'react';

import { connectGameSocket, type GameSocket, type ServerEnvelope } from '@/api/gameSocket';
import { readGuestSession } from '@/api/guestTokenManager';
import { readAccessToken } from '@/api/authTokenManager';

export type LobbyPlayer = {
  user_id: string;
  username?: string;
  score: number;
  is_drawer: boolean;
  is_online: boolean;
  is_owner: boolean;
  is_guest?: boolean;
};

export type LobbySocketState = {
  status: 'idle' | 'connecting' | 'open' | 'closed' | 'error';
  error: string;
  gameState: string; // server game_state state string
  players: LobbyPlayer[];
  minPlayers: number;
  maxPlayers: number;
};

type GameStatePayload = {
  state: string;
  players?: Array<{
    user_id: string;
    username?: string;
    score: number;
    is_drawer: boolean;
    is_online: boolean;
    is_owner?: boolean;
    is_guest?: boolean;
  }>;
  min_players?: number;
  max_players?: number;
};

function normalizePlayers(list: GameStatePayload['players'] | undefined): LobbyPlayer[] {
  if (!list) return [];
  return list.map((p) => ({
    user_id: p.user_id,
    username: p.username,
    score: p.score ?? 0,
    is_drawer: p.is_drawer ?? false,
    is_online: p.is_online ?? true,
    is_owner: p.is_owner ?? false,
    is_guest: p.is_guest ?? false,
  }));
}

export function useRoomSocket(roomId: string | undefined, inviteCode?: string) {
  const [state, setState] = useState<LobbySocketState>({
    status: 'idle',
    error: '',
    gameState: 'waiting_for_players',
    players: [],
    minPlayers: 2,
    maxPlayers: 8,
  });
  const socketRef = useRef<GameSocket | null>(null);
  // Hold a ref to the latest state/setState pair for the message callback
  // closure, so we don't re-subscribe on every state tick.
  const stateRef = useRef(state);
  stateRef.current = state;

  const send = useCallback((type: string, payload?: unknown) => {
    socketRef.current?.send(type, payload);
  }, []);

  useEffect(() => {
    if (!roomId) return;
    let cancelled = false;

    const guest = readGuestSession();
    const guestForRoom = guest && guest.roomID === roomId ? guest : null;
    const hasUser = Boolean(readAccessToken());

    // Nothing to connect with (shouldn't happen because the router guards,
    // but be safe).
    if (!hasUser && !guestForRoom) {
      setState((s) => ({ ...s, status: 'error', error: 'not authenticated' }));
      return;
    }

    setState((s) => ({ ...s, status: 'connecting', error: '' }));

    const onMessage = (env: ServerEnvelope) => {
      if (cancelled) return;
      switch (env.type) {
        case 'joined': {
          setState((s) => ({ ...s, status: 'open', error: '' }));
          break;
        }
        case 'game_state': {
          const p = env.payload as GameStatePayload;
          setState((s) => ({
            ...s,
            gameState: p.state ?? s.gameState,
            players: normalizePlayers(p.players),
            minPlayers: p.min_players ?? s.minPlayers,
            maxPlayers: p.max_players ?? s.maxPlayers,
          }));
          break;
        }
        case 'player_joined':
        case 'player_left':
        case 'player_reconnected': {
          // We'll re-derive the list from the next game_state broadcast
          // rather than maintaining partial local state — the server already
          // broadcasts game_state right after every join/leave so this is
          // just a transient no-op placeholder.
          break;
        }
        case 'error': {
          const pl = env.payload as { message?: string } | undefined;
          const msg = pl?.message ?? 'socket error';
          setState((s) => ({ ...s, error: msg }));
          break;
        }
        default:
          break;
      }
    };

    connectGameSocket({
      mode: 'private',
      roomID: roomId,
      inviteCode,
      guest: guestForRoom,
      onMessage,
      onStatusChange: (st, detail) => {
        if (cancelled) return;
        setState((s) => ({
          ...s,
          status:
            st === 'open' ? 'open' : st === 'closed' ? 'closed' : st === 'error' ? 'error' : 'connecting',
          error: detail ?? s.error,
        }));
      },
    })
      .then((sock) => {
        if (cancelled) {
          // Clean up if the effect was torn down mid-handshake.
          try {
            sock.close();
          } catch {
            /* noop */
          }
          return;
        }
        socketRef.current = sock;
      })
      .catch((err) => {
        if (cancelled) return;
        setState((s) => ({
          ...s,
          status: 'error',
          error: err instanceof Error ? err.message : 'failed to connect',
        }));
      });

    return () => {
      cancelled = true;
      if (socketRef.current) {
        try {
          socketRef.current.close();
        } catch {
          /* noop */
        }
        socketRef.current = null;
      }
    };
  }, [roomId, inviteCode]);

  // annotateOwner is kept for backwards compatibility with callers that
  // still hold the REST room.owner_id. Server-authored game_state payloads
  // now carry is_owner, so this only patches any pre-join snapshot gaps.
  const annotateOwner = useCallback((ownerID: string) => {
    if (!ownerID) return;
    setState((s) => ({
      ...s,
      players: s.players.map((p) => (p.is_owner ? p : { ...p, is_owner: p.user_id === ownerID })),
    }));
  }, []);

  return { ...state, send, annotateOwner };
}
