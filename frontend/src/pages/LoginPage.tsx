import { Link, useNavigate } from '@tanstack/react-router';
import { FormEvent, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { login, register } from '../api/auth';
import { ApiError } from '../api/http';
import { useAuthStore } from '../stores/authStore';
import { getDisplayError } from '../utils/errorMessages';

type AuthMode = 'login' | 'register';
type AuthField = 'username' | 'password' | 'confirm_password';
type AuthErrorState =
  | { kind: 'api'; error: unknown }
  | { kind: 'translation'; key: string };

const MIN_USERNAME_LENGTH = 3;
const MAX_USERNAME_LENGTH = 20;
const MIN_PASSWORD_LENGTH = 8;
const uppercaseRegex = /\p{Lu}/u;
const numberRegex = /\p{Nd}/u;
const specialCharacterRegex = /[\p{P}\p{S}]/u;

function classNames(...classes: Array<string | false | undefined>) {
  return classes.filter(Boolean).join(' ') || undefined;
}

type LoginPageProps = {
  initialMode?: AuthMode;
};

export function LoginPage({ initialMode = 'login' }: LoginPageProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setTokens = useAuthStore((state) => state.setTokens);
  const accessToken = useAuthStore((state) => state.accessToken);
  const clearTokens = useAuthStore((state) => state.clearTokens);

  const [mode, setMode] = useState<AuthMode>(initialMode);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [invalidFields, setInvalidFields] = useState<AuthField[]>([]);
  const [error, setError] = useState<AuthErrorState | null>(null);
  const [successKey, setSuccessKey] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setMode(initialMode);
    setError(null);
    setInvalidFields([]);

    if (initialMode === 'login') {
      const flashedSuccessKey = sessionStorage.getItem('drawo.auth_success');
      if (flashedSuccessKey) {
        sessionStorage.removeItem('drawo.auth_success');
        setSuccessKey(flashedSuccessKey);
        return;
      }
    }

    setSuccessKey(null);
  }, [initialMode]);

  const isRegister = mode === 'register';
  const trimmedUsernameLength = username.trim().length;
  const usernameMinLengthIsValid = trimmedUsernameLength >= MIN_USERNAME_LENGTH;
  const usernameMaxLengthIsValid = trimmedUsernameLength <= MAX_USERNAME_LENGTH;
  const usernameIsValid = usernameMinLengthIsValid && usernameMaxLengthIsValid;
  const passwordMinLengthIsValid = password.length >= MIN_PASSWORD_LENGTH;
  const passwordHasUppercase = uppercaseRegex.test(password);
  const passwordHasNumber = numberRegex.test(password);
  const passwordHasSpecialCharacter = specialCharacterRegex.test(password);
  const passwordIsValid =
    passwordMinLengthIsValid && passwordHasUppercase && passwordHasNumber && passwordHasSpecialCharacter;
  const confirmPasswordMatches = isRegister && confirmPassword.length > 0 && confirmPassword === password;

  const canSubmit = useMemo(() => {
    const baseFieldsAreReady = username.trim().length > 0 && password.trim().length > 0;
    const registerFieldsAreReady = !isRegister || (usernameIsValid && confirmPassword.trim().length > 0 && passwordIsValid);
    return baseFieldsAreReady && registerFieldsAreReady && !loading;
  }, [username, password, confirmPassword, isRegister, loading, usernameIsValid, passwordIsValid]);

  const displayError = useMemo(() => {
    if (!error) return null;
    if (error.kind === 'translation') return t(error.key);
    return getDisplayError(error.error, t('auth.fallbackError'), t);
  }, [error, t]);

  function hasInvalidField(field: AuthField) {
    return invalidFields.includes(field);
  }

  function clearInvalidField(field: AuthField) {
    setInvalidFields((fields) => fields.filter((item) => item !== field));
  }

  function markBackendErrorFields(err: unknown) {
    if (!(err instanceof ApiError)) return;

    if (err.fieldErrors) {
      const knownFields = ['username', 'password', 'confirm_password'] as const;
      setInvalidFields(knownFields.filter((field) => Boolean(err.fieldErrors?.[field])));
      return;
    }

    if (err.code === 'passwords_do_not_match') {
      setInvalidFields(['confirm_password']);
      return;
    }

    if (err.code === 'username_taken') {
      setInvalidFields(['username']);
      return;
    }

    if (!isRegister && err.code === 'invalid_credentials') {
      setInvalidFields(['username', 'password']);
    }
  }

  function resetAuthForm() {
    setError(null);
    setSuccessKey(null);
    setInvalidFields([]);
    setPassword('');
    setConfirmPassword('');
    setShowPassword(false);
    setShowConfirmPassword(false);
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setSuccessKey(null);
    setInvalidFields([]);

    try {
      if (isRegister) {
        if (password !== confirmPassword) {
          setInvalidFields(['confirm_password']);
          setError({ kind: 'translation', key: 'auth.passwordsDoNotMatch' });
          return;
        }

        await register(username.trim(), password, confirmPassword);
        sessionStorage.setItem('drawo.auth_success', 'auth.registerSuccess');
        setSuccessKey('auth.registerSuccess');
        setPassword('');
        setConfirmPassword('');
        await navigate({ to: '/login' });
        return;
      }

      const tokens = await login(username.trim(), password);
      setTokens(tokens.access_token, tokens.refresh_token);
    } catch (err) {
      markBackendErrorFields(err);
      setError({ kind: 'api', error: err });
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-shell" aria-labelledby="login-title">
        <div className="auth-hero" aria-hidden="true">
          <div className="brand-mark">D</div>
          <h1>{t('auth.title')}</h1>
          <p>{t('auth.subtitle')}</p>
          <div className="hero-canvas-preview">
            <span />
            <span />
            <span />
          </div>
        </div>

        <div className="card auth-card">
          <h2 id="login-title">
            {accessToken ? t('auth.loggedIn') : isRegister ? t('auth.register') : t('auth.login')}
          </h2>

          {accessToken ? (
            <div className="success-box" aria-live="polite">
              <p className="muted">{t('auth.loggedInBody')}</p>
              <button className="secondary-button" onClick={clearTokens} type="button">
                {t('auth.clearToken')}
              </button>
            </div>
          ) : (
            <form className="login-form" onSubmit={handleSubmit}>
              <label htmlFor="username">{t('auth.username')}</label>
              <input
                id="username"
                className={classNames(isRegister && usernameIsValid && 'is-valid', hasInvalidField('username') && 'is-error')}
                value={username}
                onChange={(event) => {
                  setUsername(event.target.value);
                  clearInvalidField('username');
                }}
                autoComplete="username"
              />

              {isRegister && (
                <ul className="field-requirements" aria-label={t('auth.usernameRequirementsTitle')}>
                  <li className={usernameMinLengthIsValid ? 'is-met' : ''}>
                    <span className="requirement-check">{usernameMinLengthIsValid ? '✓' : '○'}</span>
                    {t('auth.usernameMinLength')}
                  </li>
                  <li className={usernameMaxLengthIsValid ? 'is-met' : ''}>
                    <span className="requirement-check">{usernameMaxLengthIsValid ? '✓' : '○'}</span>
                    {t('auth.usernameMaxLength')}
                  </li>
                </ul>
              )}

              <label htmlFor="password">{t('auth.password')}</label>
              <div
                className={classNames(
                  'password-field',
                  isRegister && passwordIsValid && 'is-valid',
                  hasInvalidField('password') && 'is-error',
                )}
              >
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(event) => {
                    setPassword(event.target.value);
                    clearInvalidField('password');
                  }}
                  autoComplete={isRegister ? 'new-password' : 'current-password'}
                />
                <button
                  type="button"
                  className="password-visibility-button"
                  aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                  onClick={() => setShowPassword((value) => !value)}
                >
                  {showPassword ? '🙈' : '👁'}
                </button>
              </div>

              {isRegister && (
                <ul className="field-requirements" aria-label={t('auth.passwordRequirementsTitle')}>
                  <li className={passwordMinLengthIsValid ? 'is-met' : ''}>
                    <span className="requirement-check">{passwordMinLengthIsValid ? '✓' : '○'}</span>
                    {t('auth.passwordMinLength')}
                  </li>
                  <li className={passwordHasUppercase ? 'is-met' : ''}>
                    <span className="requirement-check">{passwordHasUppercase ? '✓' : '○'}</span>
                    {t('auth.passwordUppercase')}
                  </li>
                  <li className={passwordHasNumber ? 'is-met' : ''}>
                    <span className="requirement-check">{passwordHasNumber ? '✓' : '○'}</span>
                    {t('auth.passwordNumber')}
                  </li>
                  <li className={passwordHasSpecialCharacter ? 'is-met' : ''}>
                    <span className="requirement-check">{passwordHasSpecialCharacter ? '✓' : '○'}</span>
                    {t('auth.passwordSpecialCharacter')}
                  </li>
                </ul>
              )}

              {isRegister && (
                <>
                  <label htmlFor="confirm-password">{t('auth.confirmPassword')}</label>
                  <div
                    className={classNames(
                      'password-field',
                      confirmPasswordMatches && 'is-valid',
                      hasInvalidField('confirm_password') && 'is-error',
                    )}
                  >
                    <input
                      id="confirm-password"
                      type={showConfirmPassword ? 'text' : 'password'}
                      value={confirmPassword}
                      onChange={(event) => {
                        setConfirmPassword(event.target.value);
                        clearInvalidField('confirm_password');
                      }}
                      autoComplete="new-password"
                    />
                    <button
                      type="button"
                      className="password-visibility-button"
                      aria-label={showConfirmPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                      onClick={() => setShowConfirmPassword((value) => !value)}
                    >
                      {showConfirmPassword ? '🙈' : '👁'}
                    </button>
                  </div>
                  {confirmPasswordMatches && <p className="field-hint is-valid">✓ {t('auth.confirmPasswordMatches')}</p>}
                </>
              )}

              {displayError && (
                <p className="error-text" role="alert">
                  {displayError}
                </p>
              )}
              {successKey && (
                <p className="success-text" role="status">
                  {t(successKey)}
                </p>
              )}

              <button className="primary-button" type="submit" disabled={!canSubmit}>
                {loading
                  ? isRegister
                    ? t('auth.loadingRegister')
                    : t('auth.loadingLogin')
                  : isRegister
                    ? t('auth.register')
                    : t('auth.login')}
              </button>

              <p className="auth-switch muted">
                {isRegister ? t('auth.haveAccount') : t('auth.noAccount')}{' '}
                <Link className="link-button" onClick={resetAuthForm} to={isRegister ? '/login' : '/register'}>
                  {isRegister ? t('auth.backToLogin') : t('auth.createOne')}
                </Link>
              </p>
            </form>
          )}
        </div>
      </section>
    </main>
  );
}
