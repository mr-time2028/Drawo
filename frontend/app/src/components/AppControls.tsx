import { Link, useRouterState } from '@tanstack/react-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { isSupportedLanguage, type SupportedLanguage } from '../i18n/languages';
import { useThemeStore } from '../stores/themeStore';

export function AppControls() {
  const { t, i18n } = useTranslation();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const theme = useThemeStore((state) => state.theme);
  const toggleTheme = useThemeStore((state) => state.toggleTheme);
  const currentLanguage = isSupportedLanguage(i18n.language) ? i18n.language : 'en';
  const isPersian = currentLanguage === 'fa';
  const isDark = theme === 'dark';
  const showAuthLinks = pathname === '/';
  const [mobileMenuIsOpen, setMobileMenuIsOpen] = useState(false);

  function toggleLanguage() {
    const nextLanguage: SupportedLanguage = isPersian ? 'en' : 'fa';
    void i18n.changeLanguage(nextLanguage);
  }

  function closeMobileMenu() {
    setMobileMenuIsOpen(false);
  }

  return (
    <nav className="app-navbar" aria-label={t('common.mainNavigation')}>
      <Link className="navbar-logo" to="/" aria-label={t('common.home')} onClick={closeMobileMenu}>
        <span className="navbar-logo-mark">D</span>
        <span className="navbar-logo-text">rawo</span>
      </Link>

      <button
        aria-expanded={mobileMenuIsOpen}
        aria-label={t('common.menu')}
        className="mobile-menu-button"
        type="button"
        onClick={() => setMobileMenuIsOpen((isOpen) => !isOpen)}
      >
        <span className="mobile-menu-line" aria-hidden="true" />
        <span className="mobile-menu-line" aria-hidden="true" />
        <span className="mobile-menu-line" aria-hidden="true" />
      </button>

      <div className={`navbar-actions ${mobileMenuIsOpen ? 'is-open' : ''}`}>
        {showAuthLinks && (
          <div className="navbar-auth-links" aria-label={t('common.authLinks')}>
            <Link className="navbar-link" to="/login" onClick={closeMobileMenu}>
              {t('auth.login')}
            </Link>
            <Link className="navbar-link navbar-link-primary" to="/register" onClick={closeMobileMenu}>
              {t('auth.register')}
            </Link>
          </div>
        )}

        <div className="app-controls" aria-label={t('common.appControls')}>
          <button
            aria-checked={isPersian}
            aria-label={t('common.languageSwitch')}
            className="toggle-switch language-switch"
            role="switch"
            type="button"
            onClick={toggleLanguage}
          >
            <span className="toggle-label">{t('common.english')}</span>
            <span className="toggle-track">
              <span className="toggle-thumb" />
            </span>
            <span className="toggle-label">{t('common.persian')}</span>
          </button>

          <button
            aria-checked={isDark}
            aria-label={t('common.themeSwitch')}
            className="toggle-switch theme-switch"
            role="switch"
            type="button"
            onClick={toggleTheme}
          >
            <span className="toggle-label">☀</span>
            <span className="toggle-track">
              <span className="toggle-thumb" />
            </span>
            <span className="toggle-label">☾</span>
          </button>
        </div>
      </div>
    </nav>
  );
}
