import '@testing-library/jest-dom/vitest';
import { afterEach, vi } from 'vitest';

import { i18n } from '../i18n';
import { resetAuthStore } from '../stores/authStore';
import { resetThemeStore } from '../stores/themeStore';

Object.defineProperty(window, 'scrollTo', {
  value: vi.fn(),
  writable: true,
});

afterEach(() => {
  localStorage.clear();
  sessionStorage.clear();
  resetAuthStore();
  resetThemeStore();
  void i18n.changeLanguage('fa');
  vi.restoreAllMocks();
});
