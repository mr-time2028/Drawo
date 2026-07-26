// Centralized access to frontend environment variables.
// Keeping this in one file avoids spreading import.meta.env usage across the app.
export const env = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  defaultLanguage: import.meta.env.VITE_DEFAULT_LANGUAGE ?? 'fa',
};
