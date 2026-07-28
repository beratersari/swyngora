import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  LOCALE_META,
  SUPPORTED_LOCALES,
  isAppLocale,
  resolveAppLocale,
  type AppLocale,
} from './config';
import { setAppLocale } from './i18n';

/**
 * Flexible locale hook for switchers and settings screens.
 * Re-renders on language change via react-i18next.
 */
export function useLocale() {
  const { i18n, t } = useTranslation('common');

  const locale = useMemo(
    () => resolveAppLocale(i18n.resolvedLanguage ?? i18n.language),
    [i18n.resolvedLanguage, i18n.language],
  );

  const options = useMemo(
    () =>
      SUPPORTED_LOCALES.map((code) => ({
        value: code,
        label: t(`language.${code}`, {
          defaultValue: LOCALE_META[code]?.nativeLabel ?? code,
        }),
        meta: LOCALE_META[code],
      })),
    [t],
  );

  const setLocale = useCallback(async (next: string) => {
    const code = isAppLocale(next) ? next : resolveAppLocale(next);
    await setAppLocale(code as AppLocale);
  }, []);

  return {
    locale,
    options,
    setLocale,
    t,
    i18n,
    supportedLocales: SUPPORTED_LOCALES,
  };
}
