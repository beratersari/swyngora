import { beforeAll, describe, expect, it } from 'vitest';
import {
  DEFAULT_LOCALE,
  I18N_NAMESPACES,
  SUPPORTED_LOCALES,
  getCurrentLocale,
  hasLocaleBundle,
  initI18n,
  isAppLocale,
  listRegisteredNamespaces,
  resolveAppLocale,
  resources,
  setAppLocale,
  i18n,
} from './index';

describe('mobile i18n config', () => {
  beforeAll(() => {
    initI18n();
  });

  it('registers supported locales and namespaces', () => {
    expect(SUPPORTED_LOCALES).toContain('en');
    expect(SUPPORTED_LOCALES).toContain('tr');
    expect(listRegisteredNamespaces()).toEqual([...I18N_NAMESPACES]);
    for (const lng of SUPPORTED_LOCALES) {
      expect(hasLocaleBundle(lng)).toBe(true);
      expect(resources[lng]?.common).toBeTruthy();
    }
  });

  it('resolves locale tags', () => {
    expect(isAppLocale('en')).toBe(true);
    expect(isAppLocale('tr-TR')).toBe(true);
    expect(isAppLocale('de')).toBe(false);
    expect(resolveAppLocale('tr-TR')).toBe('tr');
    expect(resolveAppLocale('xx')).toBe(DEFAULT_LOCALE);
  });

  it('translates common keys in en and tr', async () => {
    await setAppLocale('en');
    expect(i18n.t('common:nav.markets')).toBe('Markets');
    expect(i18n.t('home:openPumps')).toMatch(/Pumps/i);

    await setAppLocale('tr');
    expect(getCurrentLocale()).toBe('tr');
    expect(i18n.t('common:nav.markets')).toBe('Piyasalar');
    expect(i18n.t('home:features')).toBe('Özellikler');

    await setAppLocale('en');
  });

  it('interpolates placeholders', async () => {
    await setAppLocale('en');
    expect(
      i18n.t('markets:summary', { count: 10, total: 100 }),
    ).toBe('Showing 10 of 100');
  });
});
