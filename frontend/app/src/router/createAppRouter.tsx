/* eslint-disable react-refresh/only-export-components */
import {
  type AnyRoute,
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from '@tanstack/react-router';

import { env } from '@/config/env';

import { AppControls } from '@/components/AppControls';
import { DevIndexPage } from '@/dev/DevIndexPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { LandingPage } from '@/pages/LandingPage';
import { LoginPage } from '@/pages/LoginPage';
import { useAuthStore } from '@/stores/authStore';

const ACCESS_TOKEN_KEY = 'drawo.access_token';

function readAccessToken(): string | null {
  // During route guards, read synchronously from both zustand and localStorage.
  // Zustand's getState() returns the current in-memory value, and localStorage
  // holds what was persisted across reloads. Either being truthy means the
  // user has a token.
  return useAuthStore.getState().accessToken ?? localStorage.getItem(ACCESS_TOKEN_KEY);
}

function RootLayout() {
  return (
    <>
      <AppControls />
      <Outlet />
    </>
  );
}

export function createAppRouter() {
  const rootRoute = createRootRoute({
    component: RootLayout,
  });

  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: LandingPage,
  });

  const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/login',
    // If the user is already authenticated, bounce them to /app BEFORE this
    // route renders. Doing the redirect in beforeLoad avoids React render/
    // effect races entirely.
    beforeLoad: () => {
      if (readAccessToken()) {
        throw redirect({ to: '/app', replace: true });
      }
    },
    component: () => <LoginPage initialMode="login" />,
  });

  const registerRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/register',
    beforeLoad: () => {
      if (readAccessToken()) {
        throw redirect({ to: '/app', replace: true });
      }
    },
    component: () => <LoginPage initialMode="register" />,
  });

  const appRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/app',
    // Anonymous users get bounced to /login BEFORE the dashboard renders.
    beforeLoad: () => {
      if (!readAccessToken()) {
        throw redirect({ to: '/login', replace: true });
      }
    },
    component: DashboardPage,
  });

  const children: AnyRoute[] = [indexRoute, loginRoute, registerRoute, appRoute];

  // Dev-only playground routes — tree-shaken in production.
  if (env.isDevelopment) {
    const devIndexRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: '/__dev__',
      beforeLoad: () => {
        if (!env.isDevelopment) throw redirect({ to: '/' });
      },
      component: DevIndexPage,
    });
    children.push(devIndexRoute);
  }

  return createRouter({ routeTree: rootRoute.addChildren(children) });
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createAppRouter>;
  }
}
