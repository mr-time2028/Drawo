import path from 'node:path';

import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// Vite is the frontend development server and bundler.
// It serves the React app on http://localhost:5173 during development.
export default defineConfig({
  // Keep one frontend project env file at frontend/.env instead of duplicating
  // another .env inside frontend/app.
  envDir: '..',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Proxy REST API + WebSocket to the Go backend on :8080 in dev. This
      // means the frontend can use same-origin URLs (/api/v1/...) and there
      // are zero CORS headaches during local development (bookmarkable
      // invite URLs opened in a fresh tab, browser extensions interfering
      // with preflight, etc.).
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
  },
});
