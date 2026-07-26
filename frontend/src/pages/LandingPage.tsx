import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

export function LandingPage() {
  const { t } = useTranslation();

  return (
    <main className="landing-page">
      <section className="landing-hero" aria-labelledby="landing-title">
        <div className="landing-copy">
          <p className="eyebrow">{t('landing.eyebrow')}</p>
          <h1 id="landing-title">
            <span>{t('landing.titleLineOne')}</span>
            <span>{t('landing.titleLineTwo')}</span>
            <span>{t('landing.titleLineThree')}</span>
          </h1>
          <p className="landing-subtitle">{t('landing.subtitle')}</p>

          <div className="landing-actions" aria-label={t('landing.actionsLabel')}>
            <Link className="primary-button landing-action" to="/register">
              {t('landing.start')}
            </Link>
          </div>
        </div>

        <div className="landing-preview" aria-label={t('landing.previewLabel')}>
          <div className="floating-drawings" aria-hidden="true">
            <div className="floating-token floating-token-balloon">
              <span className="mini-balloon-body" />
              <span className="mini-balloon-string" />
            </div>

            <div className="floating-token floating-token-car">
              <span className="mini-car-body" />
              <span className="mini-car-roof" />
              <span className="mini-car-wheel mini-car-wheel-left" />
              <span className="mini-car-wheel mini-car-wheel-right" />
            </div>

            <div className="floating-token floating-token-apple">
              <span className="mini-apple-body" />
              <span className="mini-apple-stem" />
              <span className="mini-apple-leaf" />
            </div>
          </div>

          <div className="preview-card game-preview-box">
            <div className="preview-words-box" aria-label={t('landing.previewWordsLabel')}>
              <div className="preview-word-chips">
                <button type="button" disabled>
                  {t('landing.wordBalloon')}
                </button>
                <button type="button" disabled>
                  {t('landing.wordCar')}
                </button>
                <button type="button" disabled>
                  {t('landing.wordApple')}
                </button>
              </div>
            </div>

            <div className="gameplay-steps" aria-label={t('landing.gameplayLabel')}>
              <div className="gameplay-step">
                <span>{t('landing.stepNumberOne')}</span>
                <p>{t('landing.gameplayStepOne')}</p>
              </div>
              <div className="gameplay-step">
                <span>{t('landing.stepNumberTwo')}</span>
                <p>{t('landing.gameplayStepTwo')}</p>
              </div>
              <div className="gameplay-step">
                <span>{t('landing.stepNumberThree')}</span>
                <p>{t('landing.gameplayStepThree')}</p>
              </div>
              <div className="gameplay-step">
                <span>{t('landing.stepNumberFour')}</span>
                <p>{t('landing.gameplayStepFour')}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="landing-features" aria-label={t('landing.featuresLabel')}>
        <article className="feature-card">
          <span className="feature-icon">{t('landing.featureNumberOne')}</span>
          <h2>{t('landing.featureRoomTitle')}</h2>
          <p>{t('landing.featureRoomBody')}</p>
        </article>
        <article className="feature-card">
          <span className="feature-icon">{t('landing.featureNumberTwo')}</span>
          <h2>{t('landing.featureRoundTitle')}</h2>
          <p>{t('landing.featureRoundBody')}</p>
        </article>
        <article className="feature-card">
          <span className="feature-icon">{t('landing.featureNumberThree')}</span>
          <h2>{t('landing.featureScoreTitle')}</h2>
          <p>{t('landing.featureScoreBody')}</p>
        </article>
      </section>
    </main>
  );
}
