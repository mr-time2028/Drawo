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
import { getValidAccessToken, readAccessToken, onAuthFailure } from '@/api/authTokenManager';
import { readGuestSession } from '@/api/guestTokenManager';
import { DashboardPage } from '@/pages/DashboardPage';
import { InviteLandingPage } from '@/pages/InviteLandingPage';
import { LandingPage } from '@/pages/LandingPage';
import { LoginPage } from '@/pages/LoginPage';
import { RoomLobbyPage } from '@/pages/RoomLobbyPage';

function RootLayout() {
  return (
    <div className="app-shell">
      <AppControls />
      <Outlet />
    </div>
  );
}

export function createAppRouter() {
  // Login/register guard: user already authenticated → bounce to /app.
  // beforeLoad can be async; we attempt a quick refresh to make sure the
  // session is still alive before bouncing them in. silent:true because the
  // guard handles failure by simply NOT redirecting.
  async function redirectIfAuthenticated() {
    if (!readAccessToken()) return;
    const fresh = await getValidAccessToken({ silent: true });
    if (fresh) {
      throw redirect({ to: '/app', replace: true });
    }
  }

  // Protected route guard: must have a valid token to enter /app. If the
  // token is stale we refresh once (silent — we handle the redirect here
  // ourselves to avoid double-firing the global handler during navigation).
  async function requireAuth() {
    const token = await getValidAccessToken({ silent: true });
    if (!token) {
      throw redirect({ to: '/login', replace: true });
    }
  }

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
    validateSearch: (search: Record<string, unknown>): { next?: string } => {
      const next = search.next;
      return typeof next === 'string' && next.length > 0 ? { next } : {};
    },
    beforeLoad: redirectIfAuthenticated,
    component: () => <LoginPage initialMode="login" />,
  });

  const registerRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/register',
    validateSearch: (search: Record<string, unknown>): { next?: string } => {
      const next = search.next;
      return typeof next === 'string' && next.length > 0 ? { next } : {};
    },
    beforeLoad: redirectIfAuthenticated,
    component: () => <LoginPage initialMode="register" />,
  });

  const appRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/app',
    beforeLoad: requireAuth,
    component: DashboardPage,
  });

  const inviteRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/r/$code',
    component: InviteLandingPage,
  });

  // The room lobby can be entered either as a registered user (with a valid
  // access token) or as an anonymous guest (with a guest_token issued by
  // POST /rooms/by-code/:code/join). Guests MUST also be bound to the same
  // roomId in the URL; otherwise we bounce them out.
  async function requireRoomAccess({ params }: { params: Record<string, unknown> }) {
    if (readAccessToken()) {
      const token = await getValidAccessToken({ silent: true });
      if (token) return;
    }
    const guest = readGuestSession();
    const roomId = params.roomId;
    if (guest && typeof roomId === 'string' && guest.roomID === roomId) {
      return;
    }
    throw redirect({
      to: '/login',
      replace: true,
      search: { next: typeof roomId === 'string' ? `/rooms/${roomId}` : '/' },
    });
  }

  const roomRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/rooms/$roomId',
    beforeLoad: requireRoomAccess,
    component: RoomLobbyPage,
  });

  const children: AnyRoute[] = [indexRoute, loginRoute, registerRoute, appRoute, inviteRoute, roomRoute];

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

  type AppRouter = ReturnType<typeof createRouter>;
  const router: AppRouter = createRouter({ routeTree: rootRoute.addChildren(children) });

  // Global failure handler: when refresh fails mid-session (i.e. outside a
  // navigation, for example from the axios interceptor during an API call
  // or from the proactive WS timer), navigate to /login. We use router.navigate
  // with replace:true so it's a clean transition that doesn't add to history
  // and works under jsdom/tests.
  onAuthFailure(() => {
    try {
      router.navigate({ to: '/login', replace: true });
    } catch {
      // Router may not be fully mounted yet — fall back to a hard replace.
      if (window.location.pathname !== '/login') {
        window.location.replace('/login');
      }
    }
  });

  return router;
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createAppRouter>;
  }
}
