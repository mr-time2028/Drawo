import type { TFunction } from 'i18next';

import { ApiError } from '../api/http';

function translateFieldError(field: string, message: string, t: TFunction) {
  const fieldLabel = t(`fields.${field}`);
  const translatedField = fieldLabel === `fields.${field}` ? field : fieldLabel;
  const normalizedMessageKey = message
    .toLowerCase()
    .replace(/\.$/, '')
    .replace(/ /g, '_');
  const translatedMessage = t(`validation.${normalizedMessageKey}`);
  const finalMessage =
    translatedMessage === `validation.${normalizedMessageKey}` ? message : translatedMessage;

  return `${translatedField}: ${finalMessage}`;
}

function translateKnownHttpError(error: ApiError, t: TFunction) {
  if (error.code) {
    const translated = t(`errors.${error.code}`);
    if (translated !== `errors.${error.code}`) return translated;
  }

  if (error.status === 0) {
    return t('errors.network_error');
  }

  if (error.status >= 500) {
    return t('errors.internal_server');
  }

  return null;
}

// Converts backend HTTP errors to user-facing text in the currently selected
// frontend language. It supports both regular `{message: string}` responses and
// validation responses like `{message: {password: ["Value is too short."]}}`.
export function getDisplayError(error: unknown, fallback: string, t: TFunction) {
  if (error instanceof ApiError) {
    const knownError = translateKnownHttpError(error, t);
    if (knownError) return knownError;

    if (error.fieldErrors) {
      return Object.entries(error.fieldErrors)
        .flatMap(([field, messages]) => messages.map((message) => translateFieldError(field, message, t)))
        .join('\n');
    }
  }

  return error instanceof Error ? error.message : fallback;
}
