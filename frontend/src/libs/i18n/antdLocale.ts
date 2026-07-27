import type { Locale } from 'antd/es/locale';
import enUS from 'antd/locale/en_US';
import trTR from 'antd/locale/tr_TR';
import type { AppLocale } from './config';

const ANTD_LOCALES: Record<AppLocale, Locale> = {
  en: enUS,
  tr: trTR,
};

/** Map app locale → Ant Design ConfigProvider locale pack. */
export function getAntdLocale(lng: string | null | undefined): Locale {
  const base = (lng ?? 'en').split('-')[0] ?? 'en';
  if (base in ANTD_LOCALES) {
    return ANTD_LOCALES[base as AppLocale];
  }
  return enUS;
}
