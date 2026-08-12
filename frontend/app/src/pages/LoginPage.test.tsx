import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MockAdapter from 'axios-mock-adapter';
import { afterEach, describe, expect, it } from 'vitest';

async function waitForPath(path: string) {
  await waitFor(
    () => {
      expect(window.location.pathname).toBe(path);
    },
    { timeout: 5000 },
  );
}

import { App } from '../App';
import { ApiError, httpClient } from '../api/http';
import { __resetAuthTokenManager } from '../api/authTokenManager';
import { i18n } from '../i18n';
import { useAuthStore } from '../stores/authStore';
import { getDisplayError } from '../utils/errorMessages';

// apiRequest is called with absolute `/api/v1/...` paths; baseURL is also set
// but axios-mock-adapter matches config.url which is the path passed to
// request(), so mocks MUST use the full `/api/v1/...` paths.
const API = {
  login: '/api/v1/auth/login',
  register: '/api/v1/auth/register',
  refresh: '/api/v1/auth/refresh',
  logout: '/api/v1/auth/logout',
  profile: '/api/v1/user/profile',
} as const;

const mock = new MockAdapter(httpClient, { onNoMatch: 'passthrough' });

const FAR_FUTURE_EXPIRY = 86400; // 24h

function profileMock(username: string, locale: 'en' | 'fa' = 'en') {
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

function loginResponse() {
  return { access_token: 'access-token', refresh_token: 'refresh-token', expires_in: FAR_FUTURE_EXPIRY };
}
function refreshResponse() {
  return { access_token: 'access-token-2', refresh_token: 'refresh-token-2', expires_in: FAR_FUTURE_EXPIRY };
}

async function useEnglish() {
  await i18n.changeLanguage('en');
}

afterEach(() => {
  mock.reset();
  localStorage.clear();
  sessionStorage.clear();
  useAuthStore.getState().clearTokens();
  __resetAuthTokenManager();
});

describe('LoginPage', () => {
  it('uses Persian as the default language', async () => {
    window.history.pushState({}, '', '/login');
    render(<App />);
    expect(await screen.findByRole('button', { name: 'ورود' })).toBeInTheDocument();
    expect(document.documentElement.dir).toBe('rtl');
  });

  it('logs in and redirects to the dashboard via the real router', async () => {
    await useEnglish();
    // Set up all mocks BEFORE rendering so the router beforeLoad guard (which
    // may fire a refresh) and the login/dashboard flows all see them.
    mock.onPost(API.login).reply(200, loginResponse());
    mock.onPost(API.refresh).reply(200, refreshResponse());
    mock.onGet(API.profile).reply(200, profileMock('hamid'));

    window.history.pushState({}, '', '/login');
    render(<App />);

    await userEvent.type(await screen.findByRole('textbox', { name: /username/i }), 'hamid');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /login/i }));

    expect(
      await screen.findByRole('heading', { name: /hello,\s*hamid/i }, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(window.location.pathname).toBe('/app');
    expect(localStorage.getItem('drawo.access_token')).toBe('access-token');
    expect(localStorage.getItem('drawo.refresh_token')).toBe('refresh-token');
  });

  it('registers with confirm password in the backend payload', async () => {
    await useEnglish();
    mock.onPost(API.register).reply((config) => {
      expect(JSON.parse(config.data as string)).toEqual({
        username: 'newuser',
        password: 'Secret@1',
        confirm_password: 'Secret@1',
      });
      return [201, { id: 'u1', username: 'newuser' }];
    });

    window.history.pushState({}, '', '/register');
    render(<App />);

    await userEvent.type(await screen.findByRole('textbox', { name: /username/i }), 'newuser');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'Secret@1');
    await userEvent.type(screen.getByLabelText(/confirm password/i), 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /register/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('Account created');
  });

  it('translates invalid credentials by backend error code', async () => {
    mock.onPost(API.login).reply(401, {
      message: 'invalid username or password',
      code: 'invalid_credentials',
    });

    window.history.pushState({}, '', '/login');
    render(<App />);

    const usernameInput = await screen.findByRole('textbox', { name: /نام کاربری/ });
    const passwordInput = screen.getByLabelText(/^رمز عبور$/);

    await userEvent.type(usernameInput, 'bad');
    await userEvent.type(passwordInput, 'wrong');
    await userEvent.click(screen.getByRole('button', { name: 'ورود' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('نام کاربری یا رمز عبور نامعتبر است.');
    expect(usernameInput).toHaveClass('is-error');
    expect(passwordInput.closest('.password-field')).toHaveClass('is-error');
  });

  it('translates known backend account-status error codes in Persian', async () => {
    mock.onPost(API.login).reply(403, {
      message: 'Your account has been banned.',
      code: 'account_banned',
    });

    window.history.pushState({}, '', '/login');
    render(<App />);

    await userEvent.type(await screen.findByRole('textbox', { name: /نام کاربری/ }), 'blocked');
    await userEvent.type(screen.getByLabelText(/^رمز عبور$/), 'secret');
    await userEvent.click(screen.getByRole('button', { name: 'ورود' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('حساب کاربری شما مسدود شده است');
  });

  it('switches between login/register copy and validates confirm password locally', async () => {
    await useEnglish();
    window.history.pushState({}, '', '/login');
    render(<App />);

    expect(await screen.findByRole('button', { name: /login/i })).toBeInTheDocument();

    await userEvent.click(screen.getByRole('link', { name: /create one/i }));
    expect(await screen.findByRole('button', { name: /register/i })).toBeInTheDocument();

    await userEvent.type(screen.getByRole('textbox', { name: /username/i }), 'hamid');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'Secret@1');
    const confirmInput = screen.getByLabelText(/confirm password/i);
    await userEvent.type(confirmInput, 'Different@1');
    await userEvent.click(screen.getByRole('button', { name: /register/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Passwords do not match');
    expect(confirmInput.closest('.password-field')).toHaveClass('is-error');
  });

  it('marks only the backend error field in register mode', async () => {
    await useEnglish();
    mock
      .onPost(API.register)
      .replyOnce(409, { message: { username: ['username already taken'] }, code: 'username_taken' })
      .onPost(API.register)
      .reply(201, { id: 'u1', username: 'hamid' });

    window.history.pushState({}, '', '/register');
    render(<App />);

    await userEvent.type(await screen.findByRole('textbox', { name: /username/i }), 'hamid');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'Secret@1');
    await userEvent.type(screen.getByLabelText(/confirm password/i), 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /register/i }));

    expect(await screen.findByRole('alert')).toBeInTheDocument();
    const usernameInput = screen.getByRole('textbox', { name: /username/i });
    expect(usernameInput).toHaveClass('is-error');
    expect(screen.getByLabelText(/^password$/i).closest('.password-field')).not.toHaveClass('is-error');
  });

  it('marks password or confirm-password only when backend says that exact register field failed', async () => {
    await useEnglish();
    mock.onPost(API.register).reply(400, {
      message: { confirm_password: ['Passwords do not match.'] },
      code: 'passwords_do_not_match',
    });

    window.history.pushState({}, '', '/register');
    render(<App />);

    const usernameInput = await screen.findByRole('textbox', { name: /username/i });
    const passwordInput = screen.getByLabelText(/^password$/i);
    const confirmInput = screen.getByLabelText(/confirm password/i);

    // Live requirement hints appear in English when the app is in English.
    await userEvent.type(usernameInput, 'hamid');
    await userEvent.type(passwordInput, 'short');
    expect(screen.getByText(/at least 8 characters/i).closest('li')).not.toHaveClass('is-met');
    expect(passwordInput.closest('.password-field')).not.toHaveClass('is-valid');

    await userEvent.clear(passwordInput);
    await userEvent.type(passwordInput, 'Secret@1');
    expect(screen.getByText(/at least 8 characters/i).closest('li')).toHaveClass('is-met');
    expect(screen.getByText(/at least one uppercase/i).closest('li')).toHaveClass('is-met');
    expect(screen.getByText(/at least one number/i).closest('li')).toHaveClass('is-met');
    expect(screen.getByText(/at least one special character/i).closest('li')).toHaveClass('is-met');
    expect(passwordInput.closest('.password-field')).toHaveClass('is-valid');

    // Matching confirm password shows the green "matches" hint in English.
    await userEvent.type(confirmInput, 'Secret@1');
    expect(confirmInput.closest('.password-field')).toHaveClass('is-valid');
    expect(screen.getByText(/confirm password matches/i)).toBeInTheDocument();
  });

  it('redirects anonymous users from app dashboard to login', async () => {
    window.history.pushState({}, '', '/app');
    render(<App />);
    expect(await screen.findByRole('button', { name: 'ورود' })).toBeInTheDocument();
  });

  it('shows the protected dashboard for authenticated users in Persian', async () => {
    useAuthStore.getState().setTokens({
      access_token: 'access-token',
      refresh_token: 'refresh-token',
      expires_in: FAR_FUTURE_EXPIRY,
    });
    mock.onPost(API.refresh).reply(200, refreshResponse());
    mock.onGet(API.profile).reply(200, profileMock('حمید', 'fa'));
    window.history.pushState({}, '', '/app');
    render(<App />);
    expect(await screen.findByRole('heading', { name: /سلام،?\s*حمید/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'شروع مسابقه' })).toBeInTheDocument();
  });

  it('uses global controls for language and theme on the landing page', async () => {
    window.history.pushState({}, '', '/');
    render(<App />);

    expect(await screen.findByText('بازی کن')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('switch', { name: 'تغییر زبان' }));
    expect(await screen.findByText(/^learn$/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /login/i })).toBeInTheDocument();
    expect(document.documentElement.dir).toBe('ltr');

    await userEvent.click(screen.getByRole('switch', { name: /switch theme/i }));
    expect(document.documentElement.dataset.theme).toBe('dark');
  });

  it('translates backend validation field errors to Persian', async () => {
    const fieldError = new ApiError('validation failed', 422, undefined, {
      password: ['Value is too short.', 'Must include at least one uppercase letter.'],
      confirm_password: ['Value is too short.'],
    });

    expect(getDisplayError(fieldError, 'fallback', i18n.t)).toContain(
      'رمز عبور: مقدار وارد شده خیلی کوتاه است.',
    );
    expect(getDisplayError(fieldError, 'fallback', i18n.t)).toContain(
      'رمز عبور: باید حداقل یک حرف بزرگ داشته باشد.',
    );
    expect(getDisplayError(fieldError, 'fallback', i18n.t)).toContain(
      'تکرار رمز عبور: مقدار وارد شده خیلی کوتاه است.',
    );
  });

  it('translates network and internal server errors in Persian', async () => {
    const netErr = new ApiError('Network Error', 0);
    const srvErr = new ApiError('internal server error', 500);
    expect(getDisplayError(netErr, 'fallback', i18n.t)).toContain('خطای شبکه');
    expect(getDisplayError(srvErr, 'fallback', i18n.t)).toContain('خطای داخلی سرور');
  });

  it('updates username, password requirement, and confirm password UI live in register mode', async () => {
    await useEnglish();
    window.history.pushState({}, '', '/register');
    render(<App />);

    const usernameInput = await screen.findByRole('textbox', { name: /username/i });
    const passwordInput = screen.getByLabelText(/^password$/i);

    await userEvent.type(usernameInput, 'ab');
    expect(screen.getByText(/at least 3 characters/i).closest('li')).not.toHaveClass('is-met');
    await userEvent.type(usernameInput, 'cdefghijklmnopqrstuvwxyz');
    expect(screen.getByText(/at most 20 characters/i).closest('li')).not.toHaveClass('is-met');

    await userEvent.clear(usernameInput);
    await userEvent.type(usernameInput, 'hamid');
    expect(screen.getByText(/at least 3 characters/i).closest('li')).toHaveClass('is-met');
    expect(screen.getByText(/at most 20 characters/i).closest('li')).toHaveClass('is-met');

    await userEvent.type(passwordInput, 'secret123');
    expect(passwordInput.closest('.password-field')).not.toHaveClass('is-valid');
  });

  it('does not show green password validation in login mode', async () => {
    window.history.pushState({}, '', '/login');
    render(<App />);

    const passwordInput = await screen.findByLabelText(/^رمز عبور$/);
    await userEvent.type(passwordInput, 'secret123');
    expect(passwordInput.closest('.password-field')).not.toHaveClass('is-valid');
  });

  it('toggles password visibility', async () => {
    window.history.pushState({}, '', '/login');
    render(<App />);

    const passwordInput = await screen.findByLabelText(/^رمز عبور$/);
    expect(passwordInput).toHaveAttribute('type', 'password');

    await userEvent.click(screen.getByRole('button', { name: 'نمایش رمز عبور' }));
    expect(passwordInput).toHaveAttribute('type', 'text');

    await userEvent.click(screen.getByRole('button', { name: 'مخفی کردن رمز عبور' }));
    expect(passwordInput).toHaveAttribute('type', 'password');
  });

  it('respects ?next= after a successful login (redirect back to invite)', async () => {
    await useEnglish();
    mock.onPost(API.login).reply(200, loginResponse());
    mock.onPost(API.refresh).reply(200, refreshResponse());
    // Profile may be fetched if the landing guard hits it; but /r/:code is public.
    window.history.pushState({}, '', '/login?next=%2Fr%2FABCDEF');
    render(<App />);

    await userEvent.type(await screen.findByRole('textbox', { name: /username/i }), 'hamid');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /login/i }));

    await waitForPath('/r/ABCDEF');
  });

  it('rejects unsafe (open-redirect) ?next= targets and goes to /app', async () => {
    await useEnglish();
    mock.onPost(API.login).reply(200, loginResponse());
    mock.onPost(API.refresh).reply(200, refreshResponse());
    mock.onGet(API.profile).reply(200, profileMock('hamid'));
    // Protocol-relative URL (classic open redirect) must be rejected.
    window.history.pushState({}, '', '/login?next=//evil.com/phish');
    render(<App />);
    await userEvent.type(await screen.findByRole('textbox', { name: /username/i }), 'hamid');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /login/i }));
    await waitForPath('/app');
  });

  it('logs out from the dashboard, calls the backend, clears tokens, and returns to login', async () => {
    await useEnglish();
    useAuthStore.getState().setTokens({
      access_token: 'access-token',
      refresh_token: 'refresh-token',
      expires_in: FAR_FUTURE_EXPIRY,
    });
    window.history.pushState({}, '', '/app');

    let logoutCalled = false;
    let logoutAuthHeader: string | undefined;
    mock.onPost(API.refresh).reply(200, refreshResponse());
    mock.onGet(API.profile).reply(200, profileMock('hamid'));
    mock.onPost(API.logout).reply((config) => {
      logoutCalled = true;
      logoutAuthHeader = config.headers?.Authorization as string | undefined;
      return [200, { message: 'logged out successfully' }];
    });

    render(<App />);

    await screen.findByRole('heading', { name: /hello,\s*hamid/i });
    await new Promise((r) => setTimeout(r, 50));
    await userEvent.click(screen.getByRole('button', { name: /logout/i }));

    expect(await screen.findByRole('button', { name: /login/i })).toBeInTheDocument();
    expect(window.location.pathname).toBe('/login');
    expect(localStorage.getItem('drawo.access_token')).toBeNull();
    expect(logoutCalled).toBe(true);
    expect(logoutAuthHeader).toBe('Bearer access-token');
  });
});
