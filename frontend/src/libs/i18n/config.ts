/** Supported UI languages — add a locale folder + register here to extend. */
export const SUPPORTED_LOCALES = ['en', 'tr'] as const;
export type AppLocale = (typeof SUPPORTED_LOCALES)[number];

export const DEFAULT_LOCALE: AppLocale = 'en';
export const FALLBACK_LOCALE: AppLocale = 'en';

/** i18next namespaces (one JSON file per namespace per locale). */
export const I18N_NAMESPACES = [
  'common',
  'markets',
  'detail',
  'watchlist',
  'pumps',
  'ai',
  'alerts',
  'compare',
  'signals',
  'portfolio',
  'heatmap',
  'liquidations',
  'settings',
] as const;
export type I18nNamespace = (typeof I18N_NAMESPACES)[number];

export const DEFAULT_NAMESPACE: I18nNamespace = 'common';

export const LOCALE_STORAGE_KEY = 'swyngora.locale';

export function isAppLocale(value: string | null | undefined): value is AppLocale {
  return value != null && (SUPPORTED_LOCALES as readonly string[]).includes(value);
}
