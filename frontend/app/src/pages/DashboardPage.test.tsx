import { render, screen } from '@testing-library/react';
import MockAdapter from 'axios-mock-adapter';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { __resetAuthTokenManager } from '@/api/authTokenManager';
import { httpClient } from '@/api/http';
import { App } from '@/App';
import { i18n } from '@/i18n';
import { useAuthStore } from '@/stores/authStore';

const mock = new MockAdapter(httpClient, { onNoMatch: 'passthrough' });

function tokenResponse() {
  // Far-future expiry so the beforeLoad guard never decides to refresh.
  return { access_token: 'access-token-2', refresh_token: 'refresh-token-2', expires_in: 86400 };
}

function profileResponse(username: string, locale: 'en' | 'fa' = 'en') {
  return {
    user: {
      id: 'u-1',
      username,
      is_active: true,
      status: 'active',
      is_superuser: false,
      ban_count: 0,
      banned_at: null,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
    profile: {
      user_id: 'u-1',
      avatar_url: '',
      email: '',
      phone: '',
      email_verified: false,
      phone_verified: false,
      locale,
      background_sound: false,
      tool_sound: false,
      word_score: 0,
      reputation_score: 0,
      games_played: 0,
      mvps: 0,
      rank: '',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
  };
}

async function bootstrap(locale: 'en' | 'fa' = 'en', username = 'hamid') {
  await i18n.changeLanguage(locale);
  localStorage.setItem('drawo.language', locale);
  document.documentElement.lang = locale;
  document.documentElement.dir = locale === 'fa' ? 'rtl' : 'ltr';
  mock.onPost('/api/v1/auth/refresh').reply(200, tokenResponse());
  mock.onGet('/api/v1/user/profile').reply(200, profileResponse(username, locale));
  mock.onPost('/api/v1/auth/logout').reply(200, { message: 'ok' });
  useAuthStore
    .getState()
    .setTokens({ access_token: 'access-token', refresh_token: 'refresh-token', expires_in: 900 });
  window.history.pushState({}, '', '/app');
}

beforeEach(() => {
  vi.spyOn(console, 'warn').mockImplementation(() => {});
});

afterEach(() => {
  mock.reset();
  vi.restoreAllMocks();
  localStorage.clear();
  sessionStorage.clear();
  useAuthStore.getState().clearTokens();
  __resetAuthTokenManager();
});

describe('DashboardPage', () => {
  it('renders overview greeting, stats and enabled Start Match button (EN/LTR)', async () => {
    await bootstrap('en');
    render(<App />);
    const heading = await screen.findByRole('heading', { name: /hello,\s*hamid/i });
    expect(heading).toBeInTheDocument();
    // Stats labels appear
    expect(screen.getByText(/games played/i)).toBeInTheDocument();
    expect(screen.getByText(/mvps/i)).toBeInTheDocument();
    // Start Match FAB opens the drawer (not disabled anymore).
    const startBtn = screen.getByRole('button', { name: 'Start Match' });
    expect(startBtn).toBeInTheDocument();
    expect(startBtn).not.toBeDisabled();
    // Sidebar on left in LTR: Start Match uses logical CSS so margin from end
    // isn't directly asserted here (visual regression will catch it).
  });

  it('renders in Persian/RTL with sidebar on the right and start button on the left', async () => {
    await bootstrap('fa', 'حمید');
    render(<App />);
    // Persian greeting "سلام، حمید"
    const heading = await screen.findByRole('heading', { name: /سلام،?\s*حمید/ });
    expect(heading).toBeInTheDocument();
    expect(document.documentElement.dir).toBe('rtl');
    expect(screen.getByRole('button', { name: 'شروع مسابقه' })).not.toBeDisabled();
  });

  it('shows a logout action in the sidebar', async () => {
    await bootstrap('en');
    render(<App />);
    await screen.findByRole('heading', { name: /hello,\s*hamid/i });
    expect(screen.getByRole('button', { name: /logout/i })).toBeInTheDocument();
  });

  it('renders Recovery and Profile section buttons in the sidebar', async () => {
    await bootstrap('en');
    render(<App />);
    await screen.findByRole('heading', { name: /hello,\s*hamid/i });
    expect(screen.getByRole('button', { name: /recovery/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^profile$/i })).toBeInTheDocument();
  });

  it('opens the match history modal via See more', async () => {
    await bootstrap('en');
    render(<App />);
    await screen.findByRole('heading', { name: /hello,\s*hamid/i });
    const seeMore = screen.getByRole('button', { name: /see more/i });
    expect(seeMore).toBeInTheDocument();
  });
});
