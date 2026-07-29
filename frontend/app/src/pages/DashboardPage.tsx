import { useNavigate } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { useAuthStore } from '../stores/authStore';

export function DashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const clearTokens = useAuthStore((state) => state.clearTokens);

  function handleLogout() {
    // Keep logout behavior centralized here until the backend logout endpoint is wired.
    // Clearing local tokens immediately prevents protected-route access after logout.
    clearTokens();
    void navigate({ to: '/login' });
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
            <button className="secondary-button" type="button" onClick={handleLogout}>
              {t('auth.logout')}
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
