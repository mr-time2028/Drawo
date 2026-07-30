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

  const apiBaseUrl = raw.VITE_API_BASE_URL || 'http://localhost:8080';
  // If WS URL is not provided in Docker/proxy setups, derive it from the current origin.
  const wsUrl =
    raw.VITE_WS_URL ||
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
