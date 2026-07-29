export type SupportedLanguage = 'en' | 'fa';

type LanguageMeta = {
  label: string;
  direction: 'ltr' | 'rtl';
};

// Add future languages here and create a matching JSON file in locales/.
export const languages: Record<SupportedLanguage, LanguageMeta> = {
  en: {
    label: 'English',
    direction: 'ltr',
  },
  fa: {
    label: 'فارسی',
    direction: 'rtl',
  },
};

export const defaultLanguage: SupportedLanguage = 'fa';

export function isSupportedLanguage(value: string | null | undefined): value is SupportedLanguage {
  return value === 'en' || value === 'fa';
}

export function getLanguageDirection(language: SupportedLanguage) {
  return languages[language].direction;
}
