import { useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { logout as apiLogout } from '@/api/auth';
import { useAuthStore } from '@/stores/authStore';

export function DashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const accessToken = useAuthStore((state) => state.accessToken);
  const clearTokens = useAuthStore((state) => state.clearTokens);
  const [logoutLoading, setLogoutLoading] = useState(false);

  async function handleLogout() {
    setLogoutLoading(true);
    try {
      // Call backend logout first, while we still hold the access token and
      // this component is still mounted. The access_token is always a real
      // string after the login-response normalization fix in api/auth.ts.
      if (accessToken) {
        await apiLogout(accessToken);
      }
    } catch (err) {
      // Network error / backend down: still clear local state below, but tell
      // the user that the server-side session might survive briefly.
      console.warn('[dashboard] backend logout failed:', err);
      toast.error(t('auth.logoutBackendFailed'));
    } finally {
      clearTokens();
      setLogoutLoading(false);
      void navigate({ to: '/login', replace: true });
    }
  }

  return (
    <main className="dashboard-page">
      <section className="dashboard-shell" aria-labelledby="dashboard-title">
        <div className="dashboard-hero-card">
          <p className="eyebrow">{t('dashboard.eyebrow')}</p>
          <h1 id="dashboard-title">{t('dashboard.title')}</h1>
          <p>{t('dashboard.subtitle')}</p>
          <div className="dashboard-actions">
            <button className="primary-button" type="button" disabled>
              {t('dashboard.startGame')}
            </button>
            <button
              className="secondary-button"
              type="button"
              onClick={handleLogout}
              disabled={logoutLoading}
            >
              {logoutLoading ? t('auth.loggingOut', 'Logging out…') : t('auth.logout')}
            </button>
          </div>
        </div>

        <section className="dashboard-grid" aria-label={t('dashboard.sectionsLabel')}>
          <article className="dashboard-card">
            <span className="dashboard-card-icon">✏️</span>
            <h2>{t('dashboard.playTitle')}</h2>
            <p>{t('dashboard.playBody')}</p>
          </article>
          <article className="dashboard-card">
            <span className="dashboard-card-icon">👤</span>
            <h2>{t('dashboard.profileTitle')}</h2>
            <p>{t('dashboard.profileBody')}</p>
          </article>
          <article className="dashboard-card">
            <span className="dashboard-card-icon">🤝</span>
            <h2>{t('dashboard.friendsTitle')}</h2>
            <p>{t('dashboard.friendsBody')}</p>
          </article>
          <article className="dashboard-card">
            <span className="dashboard-card-icon">⚙️</span>
            <h2>{t('dashboard.settingsTitle')}</h2>
            <p>{t('dashboard.settingsBody')}</p>
          </article>
        </section>
      </section>
    </main>
  );
}
