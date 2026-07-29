import { act, render, screen } from '@testing-library/react';
import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import userEvent from '@testing-library/user-event';
import MockAdapter from 'axios-mock-adapter';
import { afterEach, describe, expect, it } from 'vitest';

import { App } from '../App';
import { ApiError, httpClient } from '../api/http';
import { i18n } from '../i18n';
import { useAuthStore } from '../stores/authStore';
import { getDisplayError } from '../utils/errorMessages';
import { LoginPage } from './LoginPage';

const mock = new MockAdapter(httpClient);

async function useEnglish() {
  await i18n.changeLanguage('en');
}


async function renderLoginPage(initialPath: '/login' | '/register' = '/login') {
  const rootRoute = createRootRoute({ component: () => <Outlet /> });
  const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: 'login',
    component: () => <LoginPage initialMode="login" />,
  });
  const registerRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: 'register',
    component: () => <LoginPage initialMode="register" />,
  });
  const appRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: 'app',
    component: () => <p>Dashboard reached</p>,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([loginRoute, registerRoute, appRoute]),
    history: createMemoryHistory({ initialEntries: [initialPath] }),
  });

  await router.load();
  return render(<RouterProvider router={router} />);
}

afterEach(() => {
  mock.reset();
});

describe('LoginPage', () => {
  it('uses Persian as the default language', async () => {
    await renderLoginPage('/login');

    expect(screen.getByRole('button', { name: 'ورود' })).toBeInTheDocument();
    expect(document.documentElement.dir).toBe('rtl');
  });

  it('logs in and shows the logged-in state', async () => {
    await useEnglish();
    mock.onPost('/api/v1/auth/login').reply(200, {
      access_token: 'access-token',
      refresh_token: 'refresh-token',
      expires_in: 900,
    });

    await renderLoginPage('/login');

    await userEvent.type(screen.getByRole('textbox', { name: /username/i }), 'hamid');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'secret');
    await userEvent.click(screen.getByRole('button', { name: /login/i }));

    expect(await screen.findByText('Dashboard reached')).toBeInTheDocument();
    expect(localStorage.getItem('drawo.access_token')).toBe('access-token');
    expect(localStorage.getItem('drawo.refresh_token')).toBe('refresh-token');
  });

  it('registers with confirm password in the backend payload', async () => {
    await useEnglish();
    mock.onPost('/api/v1/auth/register').reply((config) => {
      expect(JSON.parse(config.data as string)).toEqual({
        username: 'newuser',
        password: 'Secret@1',
        confirm_password: 'Secret@1',
      });
      return [201, { id: 'u1', username: 'newuser' }];
    });

    await renderLoginPage('/login');

    await userEvent.click(screen.getByRole('link', { name: /create one/i }));
    await userEvent.type(screen.getByRole('textbox', { name: /username/i }), 'newuser');
    await userEvent.type(screen.getByLabelText(/^password$/i), 'Secret@1');
    await userEvent.type(screen.getByLabelText(/confirm password/i), 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /register/i }));

    expect(await screen.findByRole('status')).toHaveTextContent('Account created');
  });

  it('translates invalid credentials by backend error code', async () => {
    mock.onPost('/api/v1/auth/login').reply(401, {
      message: 'invalid username or password',
      code: 'invalid_credentials',
    });

    await renderLoginPage('/login');

    const usernameInput = screen.getByRole('textbox', { name: /نام کاربری/ });
    const passwordInput = screen.getByLabelText(/^رمز عبور$/);

    await userEvent.type(usernameInput, 'bad');
    await userEvent.type(passwordInput, 'wrong');
    await userEvent.click(screen.getByRole('button', { name: 'ورود' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('نام کاربری یا رمز عبور نامعتبر است.');
    expect(usernameInput).toHaveClass('is-error');
    expect(passwordInput.closest('.password-field')).toHaveClass('is-error');
  });

  it('translates known backend account-status error codes in Persian', async () => {
    mock.onPost('/api/v1/auth/login').reply(403, {
      message: 'Your account has been banned.',
      code: 'account_banned',
    });

    await renderLoginPage('/login');

    await userEvent.type(screen.getByRole('textbox', { name: /نام کاربری/ }), 'blocked');
    await userEvent.type(screen.getByLabelText(/^رمز عبور$/), 'secret');
    await userEvent.click(screen.getByRole('button', { name: 'ورود' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('حساب کاربری شما مسدود شده است');
  });

  it('switches between login/register copy and validates confirm password locally', async () => {
    await useEnglish();
    await renderLoginPage('/login');

    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument();

    await userEvent.click(screen.getByRole('link', { name: /create one/i }));
    expect(screen.getByRole('button', { name: /register/i })).toBeInTheDocument();

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
    mock.onPost('/api/v1/auth/register').replyOnce(409, {
      message: { username: ['username is already taken'] },
      code: 'username_taken',
    });

    await renderLoginPage('/login');

    await userEvent.click(screen.getByRole('link', { name: /create one/i }));
    const usernameInput = screen.getByRole('textbox', { name: /username/i });
    const passwordInput = screen.getByLabelText(/^password$/i);
    const confirmInput = screen.getByLabelText(/confirm password/i);

    await userEvent.type(usernameInput, 'takenuser');
    await userEvent.type(passwordInput, 'Secret@1');
    await userEvent.type(confirmInput, 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /register/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Username is already taken.');

    await act(async () => {
      await i18n.changeLanguage('fa');
    });
    expect(await screen.findByRole('alert')).toHaveTextContent('این نام کاربری قبلاً انتخاب شده است.');

    expect(usernameInput).toHaveClass('is-error');
    expect(passwordInput.closest('.password-field')).not.toHaveClass('is-error');
    expect(confirmInput.closest('.password-field')).not.toHaveClass('is-error');
  });

  it('marks password or confirm-password only when backend says that exact register field failed', async () => {
    await useEnglish();
    mock
      .onPost('/api/v1/auth/register')
      .replyOnce(422, { message: { password: ['Must include at least one special character.'] } })
      .onPost('/api/v1/auth/register')
      .replyOnce(400, { message: { confirm_password: ['passwords do not match'] }, code: 'passwords_do_not_match' });

    await renderLoginPage('/login');

    await userEvent.click(screen.getByRole('link', { name: /create one/i }));
    const usernameInput = screen.getByRole('textbox', { name: /username/i });
    const passwordInput = screen.getByLabelText(/^password$/i);
    const confirmInput = screen.getByLabelText(/confirm password/i);

    await userEvent.type(usernameInput, 'hamid');
    await userEvent.type(passwordInput, 'Secret@1');
    await userEvent.type(confirmInput, 'Secret@1');
    await userEvent.click(screen.getByRole('button', { name: /register/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Password: Must include at least one special character.');
    expect(usernameInput).not.toHaveClass('is-error');
    expect(passwordInput.closest('.password-field')).toHaveClass('is-error');
    expect(confirmInput.closest('.password-field')).not.toHaveClass('is-error');

    await userEvent.click(screen.getByRole('button', { name: /register/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Passwords do not match');
    expect(usernameInput).not.toHaveClass('is-error');
    expect(passwordInput.closest('.password-field')).not.toHaveClass('is-error');
    expect(confirmInput.closest('.password-field')).toHaveClass('is-error');
  });

  it('redirects anonymous users from app dashboard to login', async () => {
    window.history.pushState({}, '', '/app');

    render(<App />);

    expect(await screen.findByRole('button', { name: 'ورود' })).toBeInTheDocument();
  });

  it('shows the protected dashboard for authenticated users in Persian', async () => {
    useAuthStore.getState().setTokens('access-token', 'refresh-token');
    window.history.pushState({}, '', '/app');

    render(<App />);

    expect(await screen.findByRole('heading', { name: 'خوش برگشتی' })).toBeInTheDocument();
    expect(screen.getByText('شروع بازی به‌زودی')).toBeInTheDocument();
  });

  it('uses global controls for language and theme on the landing page', async () => {
    window.history.pushState({}, '', '/');
    render(<App />);

    expect(await screen.findByText('بازی کن')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('switch', { name: 'تغییر زبان' }));
    expect(screen.getByText(/^learn$/i)).toBeInTheDocument();
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

  it('translates network and internal server errors in Persian', () => {
    expect(getDisplayError(new ApiError('Network Error', 0), 'fallback', i18n.t)).toBe(
      'خطای شبکه. لطفا اتصال خود را بررسی کنید.',
    );
    expect(getDisplayError(new ApiError('internal server error', 500), 'fallback', i18n.t)).toBe(
      'خطای داخلی سرور. لطفا بعدا دوباره تلاش کنید.',
    );
  });



  it('updates username, password requirement, and confirm password UI live in register mode', async () => {
    await renderLoginPage('/login');

    await userEvent.click(screen.getByRole('link', { name: 'ثبت‌نام کنید' }));
    const usernameInput = screen.getByRole('textbox', { name: /نام کاربری/ });
    const passwordInput = screen.getByLabelText(/^رمز عبور$/);
    const confirmInput = screen.getByLabelText(/تکرار رمز عبور/);

    expect(screen.getByText(/حداقل ۳ کاراکتر/).closest('li')).not.toHaveClass('is-met');
    expect(screen.getByText(/حداکثر ۲۰ کاراکتر/).closest('li')).toHaveClass('is-met');

    await userEvent.type(usernameInput, 'ha');
    expect(usernameInput).not.toHaveClass('is-valid');
    expect(screen.getByText(/حداقل ۳ کاراکتر/).closest('li')).not.toHaveClass('is-met');

    await userEvent.type(usernameInput, 'mid');
    expect(usernameInput).toHaveClass('is-valid');
    expect(screen.getByText(/حداقل ۳ کاراکتر/).closest('li')).toHaveClass('is-met');

    await userEvent.type(usernameInput, 'abcdefghijklmnop');
    expect(usernameInput).not.toHaveClass('is-valid');
    expect(screen.getByText(/حداکثر ۲۰ کاراکتر/).closest('li')).not.toHaveClass('is-met');

    await userEvent.clear(usernameInput);
    await userEvent.type(usernameInput, 'hamid');
    expect(usernameInput).toHaveClass('is-valid');

    await userEvent.type(passwordInput, 'short');
    expect(screen.getByText(/حداقل ۸ کاراکتر/).closest('li')).not.toHaveClass('is-met');
    expect(screen.getByText(/حداقل یک حرف بزرگ/).closest('li')).not.toHaveClass('is-met');
    expect(screen.getByText(/حداقل یک عدد/).closest('li')).not.toHaveClass('is-met');
    expect(screen.getByText(/حداقل یک کاراکتر خاص/).closest('li')).not.toHaveClass('is-met');
    expect(passwordInput.closest('.password-field')).not.toHaveClass('is-valid');

    await userEvent.type(passwordInput, '123A@');
    expect(screen.getByText(/حداقل ۸ کاراکتر/).closest('li')).toHaveClass('is-met');
    expect(screen.getByText(/حداقل یک حرف بزرگ/).closest('li')).toHaveClass('is-met');
    expect(screen.getByText(/حداقل یک عدد/).closest('li')).toHaveClass('is-met');
    expect(screen.getByText(/حداقل یک کاراکتر خاص/).closest('li')).toHaveClass('is-met');
    expect(passwordInput.closest('.password-field')).toHaveClass('is-valid');

    await userEvent.type(confirmInput, 'short123A@');
    expect(confirmInput.closest('.password-field')).toHaveClass('is-valid');
    expect(screen.getByText(/تکرار رمز عبور با رمز عبور اصلی یکسان است/)).toBeInTheDocument();
  });

  it('does not show green password validation in login mode', async () => {
    await renderLoginPage('/login');

    const passwordInput = screen.getByLabelText(/^رمز عبور$/);
    await userEvent.type(passwordInput, 'secret123');

    expect(passwordInput.closest('.password-field')).not.toHaveClass('is-valid');
  });

  it('toggles password visibility', async () => {
    await renderLoginPage('/login');

    const passwordInput = screen.getByLabelText(/^رمز عبور$/);
    expect(passwordInput).toHaveAttribute('type', 'password');

    await userEvent.click(screen.getByRole('button', { name: 'نمایش رمز عبور' }));
    expect(passwordInput).toHaveAttribute('type', 'text');

    await userEvent.click(screen.getByRole('button', { name: 'مخفی کردن رمز عبور' }));
    expect(passwordInput).toHaveAttribute('type', 'password');
  });

  it('logs out from the dashboard and clears local tokens', async () => {
    await useEnglish();
    useAuthStore.getState().setTokens('access-token', 'refresh-token');
    window.history.pushState({}, '', '/app');

    render(<App />);

    expect(await screen.findByRole('heading', { name: /welcome back/i })).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /logout/i }));

    expect(await screen.findByRole('button', { name: /login/i })).toBeInTheDocument();
    expect(localStorage.getItem('drawo.access_token')).toBeNull();
  });
});
