import { Link, useRouterState } from '@tanstack/react-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { Avatar } from '@/components/ui/Avatar';
import { isSupportedLanguage, type SupportedLanguage } from '@/i18n/languages';
import { useAuthStore } from '@/stores/authStore';
import { useThemeStore } from '@/stores/themeStore';

export function AppControls() {
  const { t, i18n } = useTranslation();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const accessToken = useAuthStore((state) => state.accessToken);
  const theme = useThemeStore((state) => state.theme);
  const toggleTheme = useThemeStore((state) => state.toggleTheme);
  const currentLanguage = isSupportedLanguage(i18n.language) ? i18n.language : 'en';
  const isPersian = currentLanguage === 'fa';
  const isDark = theme === 'dark';
  const isLanding = pathname === '/';
  const isOnDashboard = pathname.startsWith('/app');
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
        {accessToken
          ? // Logged-in state: a plain circular avatar button pointing to /app.
            // When the user is already on the dashboard, hide it.
            !isOnDashboard && (
              <Link
                className="navbar-avatar-button"
                to="/app"
                aria-label={t('common.goToDashboard')}
                onClick={closeMobileMenu}
              >
                {/* Unknown-user avatar fills the button: no inner bg/border,
                    the 44px circular glass button IS the shape. */}
                <Avatar size="sm" alt="" aria-hidden className="h-full w-full text-[var(--ink)]" />
              </Link>
            )
          : // Anonymous state: login/register buttons on the landing page only.
            isLanding && (
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
            {/* Language codes are intentionally hard-coded (not translated):
                "EN" always reads "EN" and "FA" always reads "FA", regardless of
                which language is currently active. */}
            <span className="toggle-label">EN</span>
            <span className="toggle-track">
              <span className="toggle-thumb" />
            </span>
            <span className="toggle-label">FA</span>
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
