/**
 * useGameChannel — full in-game state layered on the room WebSocket.
 *
 * Extends the lobby-level socket handling with:
 *   - canvas_sync + draw op forwarding into a BoardEngine
 *   - chat/guess feed (last 100 messages, matching the server's cap)
 *   - word_suggestions / word_chosen game events for the drawer
 *   - low-latency send helpers (draw ops go out with zero batching — each op
 *     is already batched at the stroke-chunk level by the board engine)
 *
 * One hook = one socket = one engine, living for the whole room visit
 * (lobby AND game), so the canvas never misses ops during state flips.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { connectGameSocket, type GameSocket, type ServerEnvelope } from '@/api/gameSocket';
import { readAccessToken } from '@/api/authTokenManager';
import { readGuestSession } from '@/api/guestTokenManager';

import { BoardEngine } from './boardEngine';
import type { CanvasSyncPayload, DrawOp } from './drawTypes';

export type GamePlayer = {
  user_id: string;
  username?: string;
  score: number;
  is_drawer: boolean;
  is_online: boolean;
  is_owner: boolean;
  is_guest?: boolean;
  guessed_word?: boolean;
};

export type ChatEntry = {
  key: number;
  user_id?: string;
  text?: string;
  system?: boolean;
  message?: string;
  /** Marks messages the server flagged (own guess feedback etc.). */
  kind?: 'chat' | 'system' | 'correct';
};

export type WordCandidate = { group_id: string; text: string; points: number };

export type GameChannelState = {
  status: 'idle' | 'connecting' | 'open' | 'closed' | 'error';
  error: string;
  errorCode: string;
  gameState: string;
  round: number;
  maxRounds: number;
  drawerID: string;
  players: GamePlayer[];
  minPlayers: number;
  maxPlayers: number;
  endsAt: number; // unix seconds, 0 = no deadline
  wordRevealed: string; // set on round_end frames
  wordLengths: number[]; // masked blanks during drawing
  myWord: string; // drawer only: the chosen word
  suggestions: WordCandidate[]; // drawer only, during word_selection
  chat: ChatEntry[];
};

type GameStatePayload = {
  state?: string;
  round?: number;
  max_rounds?: number;
  drawer_id?: string;
  players?: GamePlayer[];
  min_players?: number;
  max_players?: number;
  ends_at?: number;
  word?: string;
  word_lengths?: number[];
};

type GameEventPayload = {
  event?: string;
  words?: WordCandidate[];
  word?: string;
  points?: number;
  group_id?: string;
};

type ChatPayload = { text?: string; system?: boolean; user_id?: string; message?: string };

const INITIAL: GameChannelState = {
  status: 'idle',
  error: '',
  errorCode: '',
  gameState: 'waiting_for_players',
  round: 0,
  maxRounds: 0,
  drawerID: '',
  players: [],
  minPlayers: 2,
  maxPlayers: 8,
  endsAt: 0,
  wordRevealed: '',
  wordLengths: [],
  myWord: '',
  suggestions: [],
  chat: [],
};

let chatKeyCounter = 0;

export function useGameChannel(roomId: string | undefined, inviteCode?: string) {
  const [state, setState] = useState<GameChannelState>(INITIAL);
  const socketRef = useRef<GameSocket | null>(null);
  // One engine per room visit (lazy state initializer keeps it stable across
  // renders without touching a ref during render).
  const [engine] = useState(() => new BoardEngine());

  useEffect(() => {
    if (!roomId) return;
    let cancelled = false;

    const guest = readGuestSession();
    const guestForRoom = guest && guest.roomID === roomId ? guest : null;
    if (!readAccessToken() && !guestForRoom) {
      // Deferred so the effect body itself never calls setState synchronously.
      queueMicrotask(() => {
        if (!cancelled) setState((s) => ({ ...s, status: 'error', error: 'not authenticated' }));
      });
      return;
    }

    queueMicrotask(() => {
      if (!cancelled) setState((s) => ({ ...s, status: 'connecting', error: '', errorCode: '' }));
    });

    const pushChat = (entry: Omit<ChatEntry, 'key'>) => {
      setState((s) => {
        const next = [...s.chat, { ...entry, key: ++chatKeyCounter }];
        return { ...s, chat: next.length > 100 ? next.slice(next.length - 100) : next };
      });
    };

    const onMessage = (env: ServerEnvelope) => {
      if (cancelled) return;
      switch (env.type) {
        case 'joined':
          setState((s) => ({ ...s, status: 'open', error: '', errorCode: '' }));
          break;
        case 'canvas_sync': {
          const p = env.payload as CanvasSyncPayload | undefined;
          engine.sync(p?.operations ?? []);
          break;
        }
        case 'draw': {
          engine.applyRemote(env.payload as DrawOp);
          break;
        }
        case 'clear_canvas': {
          engine.applyRemote({ op: 'clear' });
          break;
        }
        case 'game_state': {
          const p = (env.payload ?? {}) as GameStatePayload;
          setState((s) => ({
            ...s,
            gameState: p.state ?? s.gameState,
            round: p.round ?? s.round,
            maxRounds: p.max_rounds ?? s.maxRounds,
            drawerID: p.drawer_id ?? s.drawerID,
            players: p.players ?? s.players,
            minPlayers: p.min_players ?? s.minPlayers,
            maxPlayers: p.max_players ?? s.maxPlayers,
            endsAt: p.ends_at ?? 0,
            wordRevealed: p.word ?? '',
            wordLengths: p.word_lengths ?? [],
            // Reset the drawer's word when a new selection begins. NOTE: do
            // NOT clear `suggestions` here — the server sends word_suggestions
            // immediately BEFORE this game_state frame (startWordSelection:
            // sendWordSuggestions → broadcastGameState), so clearing would
            // erase the drawer's choices right after they arrive.
            ...(p.state === 'word_selection' || p.state === 'countdown' ? { myWord: '' } : {}),
            ...(p.state === 'drawing' || p.state === 'game_end' || p.state === 'leaderboard'
              ? { suggestions: [] }
              : {}),
          }));
          break;
        }
        case 'game': {
          const p = (env.payload ?? {}) as GameEventPayload;
          if (p.event === 'word_suggestions') {
            setState((s) => ({ ...s, suggestions: p.words ?? [] }));
          } else if (p.event === 'word_chosen') {
            setState((s) => ({ ...s, myWord: p.word ?? '', suggestions: [] }));
          }
          break;
        }
        case 'chat': {
          const p = (env.payload ?? {}) as ChatPayload;
          pushChat({
            user_id: p.user_id,
            text: p.text,
            system: p.system,
            message: p.message,
            kind: p.system ? 'system' : 'chat',
          });
          break;
        }
        case 'player_joined':
        case 'player_left':
        case 'player_reconnected':
          // game_state follows immediately; nothing to do.
          break;
        case 'error': {
          const p = env.payload as { code?: string; message?: string } | undefined;
          setState((s) => ({ ...s, error: p?.message ?? 'socket error', errorCode: p?.code ?? '' }));
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
  }, [roomId, inviteCode, engine]);

  useEffect(() => () => engine.destroy(), [engine]);

  const sendDraw = useCallback((op: DrawOp) => {
    socketRef.current?.send('draw', op);
  }, []);

  const sendChat = useCallback((text: string) => {
    const trimmed = text.trim();
    if (!trimmed) return;
    socketRef.current?.send('chat', { text: trimmed });
  }, []);

  const sendStart = useCallback(() => {
    socketRef.current?.send('game', { event: 'start' });
  }, []);

  const sendChooseWord = useCallback((groupID: string) => {
    socketRef.current?.send('game', { event: 'choose_word', group_id: groupID });
  }, []);

  const sendPlayAgain = useCallback(() => {
    socketRef.current?.send('game', { event: 'play_again' });
  }, []);

  const sendLeave = useCallback(() => {
    socketRef.current?.send('leave');
  }, []);

  const sendReport = useCallback((reportedUserID: string, reason: string, details?: string) => {
    socketRef.current?.send('game', { event: 'report', reported_user_id: reportedUserID, reason, details });
  }, []);

  const clearError = useCallback(() => {
    setState((s) => ({ ...s, error: '', errorCode: '' }));
  }, []);

  return useMemo(
    () => ({
      ...state,
      engine,
      sendDraw,
      sendChat,
      sendStart,
      sendChooseWord,
      sendPlayAgain,
      sendLeave,
      sendReport,
      clearError,
    }),
    [
      state,
      engine,
      sendDraw,
      sendChat,
      sendStart,
      sendChooseWord,
      sendPlayAgain,
      sendLeave,
      sendReport,
      clearError,
    ],
  );
}
