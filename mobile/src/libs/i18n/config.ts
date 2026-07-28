/**
 * Localization configuration — extend SUPPORTED_LOCALES + I18N_NAMESPACES
 * and add matching JSON under locales/<code>/ to grow the system.
 */

export const SUPPORTED_LOCALES = ['en', 'tr'] as const;
export type AppLocale = (typeof SUPPORTED_LOCALES)[number];

export const DEFAULT_LOCALE: AppLocale = 'en';
export const FALLBACK_LOCALE: AppLocale = 'en';

/**
 * Feature namespaces (one JSON file per namespace per locale).
 * Add a name here when introducing a new catalog file.
 */
export const I18N_NAMESPACES = [
  'common',
  'home',
  'markets',
  'watchlist',
  'pumps',
  'detail',
  'ai',
] as const;
export type I18nNamespace = (typeof I18N_NAMESPACES)[number];

export const DEFAULT_NAMESPACE: I18nNamespace = 'common';

/** Persisted language preference (web localStorage / future AsyncStorage). */
export const LOCALE_STORAGE_KEY = 'swyngora.mobile.locale.v1';

/** Optional display metadata for switchers (native names). */
export const LOCALE_META: Record<
  AppLocale,
  { nativeLabel: string; englishLabel: string }
> = {
  en: { nativeLabel: 'English', englishLabel: 'English' },
  tr: { nativeLabel: 'Türkçe', englishLabel: 'Turkish' },
};

export function isAppLocale(value: string | null | undefined): value is AppLocale {
  if (value == null || value === '') return false;
  const base = value.split('-')[0]?.toLowerCase() ?? '';
  return (SUPPORTED_LOCALES as readonly string[]).includes(base);
}

/** Normalize BCP-47 / region tags to a supported AppLocale. */
export function resolveAppLocale(value: string | null | undefined): AppLocale {
  if (!value) return DEFAULT_LOCALE;
  const base = value.split('-')[0]?.toLowerCase() ?? '';
  return isAppLocale(base) ? base : DEFAULT_LOCALE;
}
