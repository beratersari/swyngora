import { Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { isAppLocale, SUPPORTED_LOCALES, type AppLocale } from '@/libs/i18n';
import type { LanguageSwitcherProps } from './LanguageSwitcher.types';

/**
 * Locale switcher — persists via i18next language detector (localStorage).
 * Extend SUPPORTED_LOCALES + locale JSON to add languages.
 */
export function LanguageSwitcher({ className, size = 'small' }: LanguageSwitcherProps) {
  const { t, i18n } = useTranslation('common');

  const value = (i18n.resolvedLanguage?.split('-')[0] ?? i18n.language) as string;
  const current: AppLocale = isAppLocale(value) ? value : 'en';

  return (
    <Select
      className={className}
      size={size}
      value={current}
      aria-label={t('language.label')}
      style={{ minWidth: 110 }}
      options={SUPPORTED_LOCALES.map((code) => ({
        value: code,
        label: t(`language.${code}`),
      }))}
      onChange={(lng) => {
        void i18n.changeLanguage(lng);
      }}
    />
  );
}
