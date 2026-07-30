import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { env } from '../config/env';
import en from './locales/en.json';
import fa from './locales/fa.json';
import {
  defaultLanguage,
  getLanguageDirection,
  isSupportedLanguage,
  type SupportedLanguage,
} from './languages';

const LANGUAGE_STORAGE_KEY = 'drawo.language';

function getInitialLanguage(): SupportedLanguage {
  const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (isSupportedLanguage(stored)) return stored;
  if (isSupportedLanguage(env.defaultLanguage)) return env.defaultLanguage;
  return defaultLanguage;
}

export function applyDocumentLanguage(language: SupportedLanguage) {
  document.documentElement.lang = language;
  document.documentElement.dir = getLanguageDirection(language);
}

export function persistLanguage(language: SupportedLanguage) {
  localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
  applyDocumentLanguage(language);
}

i18n.use(initReactI18next).init({
  lng: getInitialLanguage(),
  fallbackLng: defaultLanguage,
  interpolation: {
    escapeValue: false,
  },
  resources: {
    en: { translation: en },
    fa: { translation: fa },
  },
});

applyDocumentLanguage(i18n.language as SupportedLanguage);

i18n.on('languageChanged', (language) => {
  if (isSupportedLanguage(language)) {
    persistLanguage(language);
  }
});

export { i18n };
