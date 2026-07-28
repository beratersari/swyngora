import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import {
  DEFAULT_LOCALE,
  DEFAULT_NAMESPACE,
  FALLBACK_LOCALE,
  I18N_NAMESPACES,
  LOCALE_STORAGE_KEY,
  SUPPORTED_LOCALES,
  resolveAppLocale,
  type AppLocale,
} from './config';
import { resources } from './resources';
import './types';

let initialized = false;

/**
 * Initialize i18n once (safe to call multiple times).
 * Detection order: storage → navigator → fallback.
 */
export function initI18n(): typeof i18n {
  if (initialized) return i18n;

  void i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      resources,
      supportedLngs: [...SUPPORTED_LOCALES],
      nonExplicitSupportedLngs: true,
      fallbackLng: FALLBACK_LOCALE,
      lng: undefined,
      defaultNS: DEFAULT_NAMESPACE,
      ns: [...I18N_NAMESPACES],
      interpolation: {
        escapeValue: false,
      },
      detection: {
        order: ['localStorage', 'navigator', 'htmlTag'],
        caches: ['localStorage'],
        lookupLocalStorage: LOCALE_STORAGE_KEY,
      },
      react: {
        useSuspense: false,
      },
      returnNull: false,
      returnEmptyString: false,
    });

  const syncDocumentLang = (lng: string) => {
    if (typeof document !== 'undefined') {
      document.documentElement.lang = resolveAppLocale(lng);
    }
  };

  i18n.on('languageChanged', syncDocumentLang);
  syncDocumentLang(i18n.language || DEFAULT_LOCALE);

  initialized = true;
  return i18n;
}

/** Current app locale (normalized). */
export function getCurrentLocale(): AppLocale {
  return resolveAppLocale(i18n.resolvedLanguage ?? i18n.language);
}

/** Change language and persist via detector cache. */
export async function setAppLocale(locale: AppLocale): Promise<void> {
  await i18n.changeLanguage(locale);
}

export { i18n };
export default i18n;
