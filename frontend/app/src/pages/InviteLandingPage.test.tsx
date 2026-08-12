/* eslint-disable @typescript-eslint/consistent-type-imports */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MockAdapter from 'axios-mock-adapter';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { httpClient } from '@/api/http';
import { __resetAuthTokenManager } from '@/api/authTokenManager';
import { i18n } from '@/i18n';
import { resetAuthStore, useAuthStore } from '@/stores/authStore';
import { InviteLandingPage } from './InviteLandingPage';

const mock = new MockAdapter(httpClient, { onNoMatch: 'passthrough' });

const FAR_FUTURE_EXPIRY = 86400;

function login() {
  useAuthStore.getState().setTokens({
    access_token: 'access-token',
    refresh_token: 'refresh-token',
    expires_in: FAR_FUTURE_EXPIRY,
  });
}

function inviteStub() {
  return {
    room_id: 'r-1',
    name: 'Fun Room',
    state: 'lobby',
    language: 'en',
    word_source: 'default',
    has_password: false,
    max_players: 8,
    min_players: 2,
    round_time: 80,
    max_rounds: 3,
    custom_category_count: 0,
    custom_word_count: 0,
    invite_code: 'ABCDEF',
  };
}

// Minimal stub: InviteLandingPage uses useParams/useNavigate from
// @tanstack/react-router. We stub them via vi.mock at module load.
const navigate = vi.fn();
let params: Record<string, string | undefined> = { code: 'ABCDEF' };

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual<typeof import('@tanstack/react-router')>('@tanstack/react-router');
  return {
    ...actual,
    useParams: () => params,
    useNavigate: () => navigate,
    Link: ({
      to,
      children,
      search,
      ...rest
    }: {
      to?: string;
      children?: React.ReactNode;
      search?: Record<string, string>;
    } & Record<string, unknown>) => (
      <span
        data-link-to={
          typeof to === 'string' ? to + (search ? `?${new URLSearchParams(search).toString()}` : '') : ''
        }
        {...rest}
      >
        {children}
      </span>
    ),
  };
});

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

beforeEach(async () => {
  await i18n.changeLanguage('en');
  params = { code: 'ABCDEF' };
  navigate.mockReset();
});

afterEach(() => {
  mock.reset();
  localStorage.clear();
  sessionStorage.clear();
  resetAuthStore();
  __resetAuthTokenManager();
});

describe('InviteLandingPage', () => {
  it('shows loading state then room info; anonymous users get login CTA', async () => {
    mock.onGet('/api/v1/rooms/by-code/ABCDEF').reply(200, inviteStub());
    render(<InviteLandingPage />);
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
    expect(await screen.findByRole('heading', { name: 'Fun Room' })).toBeInTheDocument();
    const loginBtn = screen.getByText(/log in to join/i).closest('[data-link-to]');
    expect(loginBtn).not.toBeNull();
    expect(loginBtn).toHaveAttribute('data-link-to', '/login?next=%2Fr%2FABCDEF');
  });

  it('shows invalid invite state when code is missing', async () => {
    params = {};
    render(<InviteLandingPage />);
    expect(await screen.findByRole('heading', { name: /invalid invite/i })).toBeInTheDocument();
    expect(screen.getByText(/back to dashboard/i)).toBeInTheDocument();
  });

  it('shows error state when backend 404s', async () => {
    mock.onGet('/api/v1/rooms/by-code/BAD').reply(404, { message: 'not found' });
    params = { code: 'BAD' };
    render(<InviteLandingPage />);
    expect(await screen.findByRole('heading', { name: /invalid invite/i })).toBeInTheDocument();
  });

  it('joins a password-less room automatically when logged in', async () => {
    login();
    mock.onGet('/api/v1/rooms/by-code/ABCDEF').reply(200, inviteStub());
    mock.onPost('/api/v1/rooms/by-code/ABCDEF/join').reply(200, { id: 'r-1', ...inviteStub() });
    render(<InviteLandingPage />);
    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith({ to: '/rooms/r-1', replace: true });
    });
  });

  it('prompts for password when the room requires one', async () => {
    login();
    const withPw = { ...inviteStub(), has_password: true };
    mock.onGet('/api/v1/rooms/by-code/ABCDEF').reply(200, withPw);
    mock.onPost('/api/v1/rooms/by-code/ABCDEF/join').reply(200, { id: 'r-1', ...withPw });

    render(<InviteLandingPage />);
    const pwInput = await screen.findByLabelText(/password/i);
    expect(pwInput).toBeInTheDocument();
    await userEvent.type(pwInput, 'secret');
    await userEvent.click(screen.getByRole('button', { name: /join room/i }));
    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith({ to: '/rooms/r-1', replace: true });
    });
  });

  it('disables join for non-lobby rooms when user is logged in', async () => {
    login();
    const started = { ...inviteStub(), state: 'playing' };
    mock.onGet('/api/v1/rooms/by-code/ABCDEF').reply(200, started);
    render(<InviteLandingPage />);
    expect(await screen.findByRole('heading', { name: 'Fun Room' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /game in progress/i })).toBeDisabled();
  });

  it('shows a toast and returns to ready state on non-password join error', async () => {
    login();
    mock.onGet('/api/v1/rooms/by-code/ABCDEF').reply(200, inviteStub());
    mock.onPost('/api/v1/rooms/by-code/ABCDEF/join').reply(500, { message: 'server exploded' });
    const { toast } = await import('sonner');
    render(<InviteLandingPage />);
    // Auto-join fails with non-password error → error toast shown, button returns to ready.
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled();
    });
    expect(await screen.findByRole('button', { name: /join room/i })).toBeInTheDocument();
  });
});
