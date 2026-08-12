/**
 * Centralized, typed access to frontend environment variables.
 *
 * Keeping this in one file avoids spreading `import.meta.env` usage across the
 * app, and ensures Vite_* variables are validated/typed at dev/build time.
 */
type FrontendEnv = {
  apiBaseUrl: string;
  wsUrl: string;
  defaultLanguage: 'en' | 'fa';
  isDevelopment: boolean;
  appVersion: string;
};

function readEnv(): FrontendEnv {
  const raw = import.meta.env;

  // Default: same-origin. In `npm run dev` this hits the Vite proxy
  // (configured below in vite.config.ts → http://localhost:8080). In Docker
  // or production Nginx, /api is reverse-proxied at the edge. Override with
  // VITE_API_BASE_URL only if you need to point at a different host.
  const apiBaseUrl = raw.VITE_API_BASE_URL || '';
  // If WS URL is not provided, derive it from the current origin — this
  // matches the dev proxy and production Nginx paths. If VITE_API_BASE_URL
  // is set explicitly, derive WS from that too.
  const explicit = typeof raw.VITE_API_BASE_URL === 'string' && raw.VITE_API_BASE_URL ? raw.VITE_API_BASE_URL : '';
  const derivedFromApi = explicit
    ? `${explicit.replace(/^http/, 'ws')}/api/v1/ws`
    : null;
  const wsUrl =
    raw.VITE_WS_URL ||
    derivedFromApi ||
    (typeof window !== 'undefined'
      ? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/api/v1/ws`
      : 'ws://localhost:8080/api/v1/ws');

  const rawLang = (raw.VITE_DEFAULT_LANGUAGE || 'fa').toLowerCase();
  const defaultLanguage: 'en' | 'fa' = rawLang === 'en' ? 'en' : 'fa';

  return {
    apiBaseUrl,
    wsUrl,
    defaultLanguage,
    isDevelopment: raw.DEV === true,
    appVersion: raw.VITE_APP_VERSION || '0.1.0',
  };
}

export const env: FrontendEnv = readEnv();
