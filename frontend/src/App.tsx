import {
  Outlet,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import { useState } from 'react';

import { AppControls } from './components/AppControls';
import { LandingPage } from './pages/LandingPage';
import { LoginPage } from './pages/LoginPage';

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
    path: 'login',
    component: () => <LoginPage initialMode="login" />,
  });

  const registerRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: 'register',
    component: () => <LoginPage initialMode="register" />,
  });

  return createRouter({ routeTree: rootRoute.addChildren([indexRoute, loginRoute, registerRoute]) });
}

const router = createAppRouter();

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export function App() {
  const [appRouter] = useState(() => createAppRouter());

  return <RouterProvider router={appRouter} />;
}
