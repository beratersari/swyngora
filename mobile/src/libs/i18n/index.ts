export { i18n, initI18n, getCurrentLocale, setAppLocale, default } from './i18n';
export {
  SUPPORTED_LOCALES,
  DEFAULT_LOCALE,
  FALLBACK_LOCALE,
  I18N_NAMESPACES,
  DEFAULT_NAMESPACE,
  LOCALE_STORAGE_KEY,
  LOCALE_META,
  isAppLocale,
  resolveAppLocale,
  type AppLocale,
  type I18nNamespace,
} from './config';
export {
  resources,
  hasLocaleBundle,
  listRegisteredNamespaces,
  type AppResources,
} from './resources';
export { useLocale } from './useLocale';
