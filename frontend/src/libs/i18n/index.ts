import './types';
export { i18n, default } from './i18n';
export {
  SUPPORTED_LOCALES,
  DEFAULT_LOCALE,
  FALLBACK_LOCALE,
  I18N_NAMESPACES,
  LOCALE_STORAGE_KEY,
  isAppLocale,
  type AppLocale,
  type I18nNamespace,
} from './config';
export { getAntdLocale } from './antdLocale';
export { resources, type AppResources } from './resources';
