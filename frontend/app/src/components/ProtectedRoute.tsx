import { Navigate } from '@tanstack/react-router';
import { ReactNode } from 'react';

import { useAuthStore } from '../stores/authStore';

type ProtectedRouteProps = {
  children: ReactNode;
};

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const accessToken = useAuthStore((state) => state.accessToken);

  // The backend remains the source of truth, but the route guard prevents
  // obviously anonymous users from seeing protected UI during normal navigation.
  if (!accessToken) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}
