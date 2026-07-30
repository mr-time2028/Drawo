/* eslint-disable react-refresh/only-export-components */
import { RouterProvider } from '@tanstack/react-router';
import { useState } from 'react';

import { createAppRouter } from './createAppRouter';

export function App() {
  const [appRouter] = useState(() => createAppRouter());

  return <RouterProvider router={appRouter} />;
}

export { createAppRouter };
