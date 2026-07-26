import { create } from 'zustand';

type Theme = 'light' | 'dark';

const THEME_STORAGE_KEY = 'drawo.theme';

function isTheme(value: string | null): value is Theme {
  return value === 'light' || value === 'dark';
}

function getInitialTheme(): Theme {
  const stored = localStorage.getItem(THEME_STORAGE_KEY);
  if (isTheme(stored)) return stored;
  return 'light';
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
}

type ThemeState = {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (theme: Theme) => void;
};

export const useThemeStore = create<ThemeState>((set, get) => ({
  theme: getInitialTheme(),

  setTheme: (theme) => {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
    applyTheme(theme);
    set({ theme });
  },

  toggleTheme: () => {
    const nextTheme = get().theme === 'light' ? 'dark' : 'light';
    get().setTheme(nextTheme);
  },
}));

applyTheme(getInitialTheme());

export function resetThemeStore() {
  useThemeStore.setState({ theme: 'light' });
  applyTheme('light');
}
