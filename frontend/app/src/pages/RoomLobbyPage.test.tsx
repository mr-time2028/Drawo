import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import MockAdapter from 'axios-mock-adapter';
import { type ReactNode, useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

/* eslint-disable @typescript-eslint/consistent-type-imports */

import { httpClient } from '@/api/http';
import { __resetAuthTokenManager } from '@/api/authTokenManager';
import { i18n } from '@/i18n';
import { resetAuthStore } from '@/stores/authStore';
import { RoomLobbyPage } from './RoomLobbyPage';

const mock = new MockAdapter(httpClient, { onNoMatch: 'passthrough' });

const navigate = vi.fn();
let params: Record<string, string | undefined> = { roomId: 'r-1' };

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual<typeof import('@tanstack/react-router')>('@tanstack/react-router');
  return {
    ...actual,
    useParams: () => params,
    useNavigate: () => navigate,
    Link: ({ children, ...rest }: { to?: string; children?: ReactNode } & Record<string, unknown>) => (
      <span data-link {...rest}>
        {children}
      </span>
    ),
  };
});

// Stub the WebSocket hook so lobby unit tests don't attempt real connections.
// Tests that care about socket-driven state can push updates via setSocketState.
let setSocketState: (patch: Record<string, unknown>) => void;
const sendMock = vi.fn();
const annotateOwnerMock = vi.fn();

vi.mock('@/api/useRoomSocket', () => ({
  useRoomSocket: () => {
    const [state, setState] = useState({
      status: 'open',
      error: '',
      gameState: 'waiting_for_players',
      players: [],
      minPlayers: 2,
      maxPlayers: 8,
    });
    setSocketState = (patch) => setState((s: any) => ({ ...s, ...patch }));
    return { ...state, send: sendMock, annotateOwner: annotateOwnerMock };
  },
}));

const writeText = vi.fn(() => Promise.resolve());
Object.assign(navigator, {
  clipboard: { writeText },
});

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

const roomStub = {
  id: 'r-1',
  name: 'My Room',
  invite_code: 'ABCDEF',
  owner_id: 'u-1',
  type: 'private',
  has_password: false,
  language: 'en',
  word_source: 'default',
  state: 'lobby',
  min_players: 2,
  max_players: 8,
  round_time: 80,
  max_rounds: 3,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const profileStub = {
  user: { id: 'u-1', username: 'hamid' },
  profile: { locale: 'en' },
};

beforeEach(async () => {
  await i18n.changeLanguage('en');
  params = { roomId: 'r-1' };
  navigate.mockReset();
  writeText.mockReset();
  sendMock.mockReset();
  annotateOwnerMock.mockReset();
  mock.reset();
});

afterEach(() => {
  mock.reset();
  localStorage.clear();
  sessionStorage.clear();
  resetAuthStore();
  __resetAuthTokenManager();
});

describe('RoomLobbyPage', () => {
  it('shows loading and then room metadata; owner sees start button', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, profileStub);
    render(<RoomLobbyPage />);
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    expect(screen.getByText(/private room/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /start match/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /leave room/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /close room/i })).toBeInTheDocument();
  });

  it('non-owner sees waiting message instead of start', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock
      .onGet('/api/v1/user/profile')
      .reply(200, { user: { id: 'u-2', username: 'friend' }, profile: { locale: 'en' } });
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /waiting for owner/i })).toBeDisabled();
    expect(screen.queryByRole('button', { name: /close room/i })).not.toBeInTheDocument();
  });

  it('copies invite link to clipboard', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, profileStub);
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /copy link/i }));
    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(`${window.location.origin}/r/ABCDEF`);
    });
    expect(await screen.findByRole('button', { name: /copied/i })).toBeInTheDocument();
  });

  it('leave button calls POST /leave and navigates back', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, { ...profileStub, user: { id: 'u-2', username: 'f' } });
    mock.onPost('/api/v1/rooms/r-1/leave').reply(200, roomStub);
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /leave room/i }));
    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith({ to: '/app', replace: true });
    });
    expect(mock.history.post.find((r) => r.url?.includes('/leave'))).toBeTruthy();
  });

  it('close button (owner) calls POST /close and navigates back', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, profileStub);
    mock.onPost('/api/v1/rooms/r-1/close').reply(200, { message: 'ok' });
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /close room/i }));
    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith({ to: '/app', replace: true });
    });
  });

  it('start button sends WS game "start" event and flips to in-progress state', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, profileStub);
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    // Simulate a second player joining so the Start button enables (needs ≥2).
    setSocketState({
      players: [
        { user_id: 'u-1', username: 'hamid', score: 0, is_drawer: false, is_online: true, is_owner: true, is_guest: false },
        { user_id: 'u-2', username: 'friend', score: 0, is_drawer: false, is_online: true, is_owner: false, is_guest: false },
      ],
    });
    const startBtn = await screen.findByRole('button', { name: /start match/i });
    await waitFor(() => expect(startBtn).not.toBeDisabled());
    fireEvent.click(startBtn);
    await waitFor(() => {
      expect(sendMock).toHaveBeenCalledWith('game', { event: 'start' });
    });
    // Simulate the hub flipping to countdown.
    setSocketState({ gameState: 'countdown' });
    expect(await screen.findByRole('button', { name: /game in progress/i })).toBeInTheDocument();
  });

  it('navigates back when room fails to load', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(404, { message: 'nope' });
    mock.onGet('/api/v1/user/profile').reply(200, profileStub);
    render(<RoomLobbyPage />);
    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith({ to: '/', replace: true });
    });
  });

  it('renders without roomId param (shows loading then nothing)', async () => {
    params = {};
    render(<RoomLobbyPage />);
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('shows error toast when clipboard copy fails', async () => {
    writeText.mockRejectedValueOnce(new Error('not allowed'));
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, profileStub);
    const { toast } = await import('sonner');
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /copy link/i }));
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled();
    });
  });

  it('shows error toast when leave fails', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, { ...profileStub, user: { id: 'u-2', username: 'f' } });
    mock.onPost('/api/v1/rooms/r-1/leave').reply(500, { message: 'boom' });
    const { toast } = await import('sonner');
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /leave room/i }));
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled();
    });
  });

  it('shows error toast when close fails', async () => {
    mock.onGet('/api/v1/rooms/r-1').reply(200, roomStub);
    mock.onGet('/api/v1/user/profile').reply(200, profileStub);
    mock.onPost('/api/v1/rooms/r-1/close').reply(403, { message: 'forbidden' });
    const { toast } = await import('sonner');
    render(<RoomLobbyPage />);
    expect(await screen.findByRole('heading', { name: 'My Room' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /close room/i }));
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled();
    });
  });
});
