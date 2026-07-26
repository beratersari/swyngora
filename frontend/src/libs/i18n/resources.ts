import commonEn from './locales/en/common.json';
import marketsEn from './locales/en/markets.json';
import detailEn from './locales/en/detail.json';
import commonTr from './locales/tr/common.json';
import marketsTr from './locales/tr/markets.json';
import detailTr from './locales/tr/detail.json';

/**
 * Bundled translation catalogs.
 * To add a language: create `locales/<code>/*.json` and register here + in config.ts.
 */
export const resources = {
  en: {
    common: commonEn,
    markets: marketsEn,
    detail: detailEn,
  },
  tr: {
    common: commonTr,
    markets: marketsTr,
    detail: detailTr,
  },
} as const;

export type AppResources = (typeof resources)['en'];
