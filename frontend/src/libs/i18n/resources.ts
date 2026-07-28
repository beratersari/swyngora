import commonEn from './locales/en/common.json';
import marketsEn from './locales/en/markets.json';
import detailEn from './locales/en/detail.json';
import watchlistEn from './locales/en/watchlist.json';
import pumpsEn from './locales/en/pumps.json';
import commonTr from './locales/tr/common.json';
import marketsTr from './locales/tr/markets.json';
import detailTr from './locales/tr/detail.json';
import watchlistTr from './locales/tr/watchlist.json';
import pumpsTr from './locales/tr/pumps.json';

/**
 * Bundled translation catalogs.
 * To add a language: create `locales/<code>/*.json` and register here + in config.ts.
 */
export const resources = {
  en: {
    common: commonEn,
    markets: marketsEn,
    detail: detailEn,
    watchlist: watchlistEn,
    pumps: pumpsEn,
  },
  tr: {
    common: commonTr,
    markets: marketsTr,
    detail: detailTr,
    watchlist: watchlistTr,
    pumps: pumpsTr,
  },
} as const;

export type AppResources = (typeof resources)['en'];
