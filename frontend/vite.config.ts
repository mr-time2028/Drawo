import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// Vite is the frontend development server and bundler.
// It serves the React app on http://localhost:5173 during development.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
  },
});
