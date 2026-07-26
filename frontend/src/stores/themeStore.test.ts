import { describe, expect, it } from 'vitest';

import { useThemeStore } from './themeStore';

describe('themeStore', () => {
  it('toggles theme and updates document attribute', () => {
    useThemeStore.getState().setTheme('light');
    expect(document.documentElement.dataset.theme).toBe('light');

    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');
    expect(localStorage.getItem('drawo.theme')).toBe('dark');
  });
});
