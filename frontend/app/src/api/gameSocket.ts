/**
 * Minimal WebSocket client for the Drawo realtime server.
 *
 * Supports authenticating as either:
 *   1. A registered user (Bearer access_token), or
 *   2. An anonymous guest (guest_token, room-bound, issued by joinRoomByCode).
 *
 * The socket lifecycle is:
 *   connect() → sends `auth` frame → receives `auth_ok` → sends `join` frame → receives `joined`.
 *
 * This module is intentionally focused on wiring up the guest/registered
 * connection flow — full game events (draw, chat, game state) will be layered
 * on top as the in-game page is built.
 */

import { env } from '@/config/env';
import { useAuthStore } from '@/stores/authStore';
import { readGuestSession } from './guestTokenManager';
import { getValidAccessToken, wsAcquireToken, wsReleaseToken } from './authTokenManager';

export type SocketStatus = 'idle' | 'connecting' | 'authenticating' | 'joining' | 'open' | 'closed' | 'error';

export type ServerEnvelope = {
  type: string;
  payload?: unknown;
  seq?: number;
  timestamp?: number;
};

type AuthMode =
  { kind: 'user' } | { kind: 'guest'; guestToken: string; roomID: string; nickname: string; guestID: string };

export type ConnectOptions = {
  mode?: 'public' | 'private' | 'reconnect';
  roomID?: string;
  inviteCode?: string;
  language?: string;
  categoryID?: string;
  /** If provided, auth as a guest with this session instead of sniffing from localStorage. */
  guest?: { guestToken: string; guestID: string; nickname: string; roomID: string } | null;
  onMessage?: (env: ServerEnvelope) => void;
  onStatusChange?: (status: SocketStatus, detail?: string) => void;
};

export type GameSocket = {
  close: () => void;
  send: (type: string, payload?: unknown) => void;
  getStatus: () => SocketStatus;
};

export async function connectGameSocket(opts: ConnectOptions): Promise<GameSocket> {
  let closedByUser = false;
  let status: SocketStatus = 'idle';
  let ws: WebSocket | null = null;

  const setStatus = (s: SocketStatus, detail?: string) => {
    status = s;
    opts.onStatusChange?.(s, detail);
  };

  // Decide auth mode. An explicit opts.guest always wins (caller already
  // validated room-binding); otherwise fall back to the localStorage guest
  // session only if it matches the target room.
  let authMode: AuthMode;
  const storedGuest = readGuestSession();
  if (opts.guest) {
    authMode = { kind: 'guest', ...opts.guest };
  } else if (storedGuest && (!opts.roomID || opts.roomID === storedGuest.roomID)) {
    authMode = { kind: 'guest', ...storedGuest };
  } else {
    authMode = { kind: 'user' };
  }

  setStatus('connecting');

  return new Promise<GameSocket>((resolve, reject) => {
    const socket = new WebSocket(env.wsUrl);
    ws = socket;
    socket.binaryType = 'arraybuffer';

    let unsubscribeTokenSync: (() => void) | null = null;

    const cleanup = () => {
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      unsubscribeTokenSync?.();
      unsubscribeTokenSync = null;
      if (authMode.kind === 'user') {
        wsReleaseToken();
      }
    };

    const send = (type: string, payload?: unknown) => {
      if (socket.readyState !== WebSocket.OPEN) return;
      socket.send(
        JSON.stringify({
          type,
          payload: payload ?? undefined,
          timestamp: Date.now(),
        }),
      );
    };

    // Keep the socket authenticated for as long as it lives: whenever the
    // token manager rotates the access token (the proactive refresh timer
    // fires ~60s before expiry while a socket is open), immediately re-auth
    // over the SAME connection. Combined with the server's `auth_required`
    // nudge below, a user idling in a lobby is never kicked by token expiry.
    if (authMode.kind === 'user') {
      let lastSentToken: string | null = null;
      unsubscribeTokenSync = useAuthStore.subscribe((state) => {
        const token = state.accessToken;
        if (!token || token === lastSentToken) return;
        if (socket.readyState === WebSocket.OPEN) {
          lastSentToken = token;
          send('auth', { access_token: token });
        }
      });
    }

    let settled = false;
    const fail = (msg: string) => {
      if (settled) return;
      settled = true;
      cleanup();
      try {
        socket.close();
      } catch {
        /* noop */
      }
      setStatus('error', msg);
      reject(new Error(msg));
    };

    const doJoin = () => {
      setStatus('joining');
      let mode = opts.mode ?? 'public';
      let roomID = opts.roomID;
      const inviteCode = opts.inviteCode;

      if (authMode.kind === 'guest') {
        // Guests are room-bound — always join their specific room.
        mode = 'private';
        roomID = authMode.roomID;
      }

      send('join', {
        mode,
        room_id: roomID,
        invite_code: inviteCode,
        language: opts.language,
        category_id: opts.categoryID,
      });
    };

    socket.onopen = async () => {
      setStatus('authenticating');
      if (authMode.kind === 'guest') {
        send('auth', { guest_token: authMode.guestToken });
      } else {
        try {
          const token = await wsAcquireToken();
          if (!token) {
            fail('not authenticated');
            return;
          }
          send('auth', { access_token: token });
        } catch (err) {
          fail(err instanceof Error ? err.message : 'auth failed');
        }
      }
    };

    socket.onmessage = (ev) => {
      if (typeof ev.data !== 'string') return;
      let env: ServerEnvelope;
      try {
        env = JSON.parse(ev.data) as ServerEnvelope;
      } catch {
        return;
      }

      opts.onMessage?.(env);

      switch (env.type) {
        case 'auth_ok':
          if (!settled) {
            // Still in handshake — next frame must be join.
            doJoin();
          }
          break;
        case 'auth_required':
          // Server warns the access token is close to expiry. Refresh in the
          // background and re-auth over the SAME socket — the user must never
          // notice (no reconnect, no seat loss). Guests can't refresh; their
          // 24h token comfortably outlives any match.
          if (authMode.kind === 'user') {
            void getValidAccessToken({ silent: true }).then((token) => {
              if (token && socket.readyState === WebSocket.OPEN) {
                send('auth', { access_token: token });
              }
            });
          }
          break;
        case 'joined':
          setStatus('open');
          if (!settled) {
            settled = true;
            resolve({
              close: () => {
                closedByUser = true;
                try {
                  send('leave');
                } catch {
                  /* noop */
                }
                cleanup();
                try {
                  socket.close();
                } catch {
                  /* noop */
                }
                setStatus('closed');
              },
              send,
              getStatus: () => status,
            });
          }
          break;
        case 'error': {
          const payload = env.payload as { code?: string; message?: string } | undefined;
          const msg = payload?.message ?? 'socket error';
          if (!settled) {
            fail(msg);
          }
          break;
        }
        default:
          // All other messages (chat, draw, game_state, player_joined, etc.)
          // are forwarded to the caller via onMessage.
          break;
      }
    };

    socket.onerror = () => {
      if (!settled) fail('websocket connection failed');
    };

    socket.onclose = () => {
      cleanup();
      setStatus('closed');
      if (!settled && !closedByUser) {
        fail('websocket closed before handshake completed');
      }
    };
  });
}

/**
 * Returns the saved guest identity suitable for passing as `opts.guest` to
 * connectGameSocket, or null when the user has no guest session (or it is
 * bound to a different room).
 */
export function getGuestAuthForRoom(roomID: string) {
  const g = readGuestSession();
  if (!g) return null;
  if (g.roomID !== roomID) return null;
  return g;
}
