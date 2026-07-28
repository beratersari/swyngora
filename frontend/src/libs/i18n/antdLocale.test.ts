import { describe, expect, it } from 'vitest';
import enUS from 'antd/locale/en_US';
import trTR from 'antd/locale/tr_TR';
import { getAntdLocale } from './antdLocale';
import { isAppLocale, SUPPORTED_LOCALES } from './config';

describe('isAppLocale', () => {
  it('accepts supported codes', () => {
    for (const code of SUPPORTED_LOCALES) {
      expect(isAppLocale(code)).toBe(true);
    }
  });

  it('rejects unknown or regional codes without split', () => {
    expect(isAppLocale('de')).toBe(false);
    expect(isAppLocale('en-US')).toBe(false);
    expect(isAppLocale(null)).toBe(false);
    expect(isAppLocale(undefined)).toBe(false);
  });
});

describe('getAntdLocale', () => {
  it('maps en and tr (with region)', () => {
    expect(getAntdLocale('en')).toBe(enUS);
    expect(getAntdLocale('tr')).toBe(trTR);
    expect(getAntdLocale('tr-TR')).toBe(trTR);
  });

  it('falls back to en for unknown', () => {
    expect(getAntdLocale('de')).toBe(enUS);
    expect(getAntdLocale(undefined)).toBe(enUS);
  });
});
