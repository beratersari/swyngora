import commonEn from './locales/en/common.json';
import marketsEn from './locales/en/markets.json';
import detailEn from './locales/en/detail.json';
import watchlistEn from './locales/en/watchlist.json';
import pumpsEn from './locales/en/pumps.json';
import aiEn from './locales/en/ai.json';
import alertsEn from './locales/en/alerts.json';
import compareEn from './locales/en/compare.json';
import signalsEn from './locales/en/signals.json';
import commonTr from './locales/tr/common.json';
import marketsTr from './locales/tr/markets.json';
import detailTr from './locales/tr/detail.json';
import watchlistTr from './locales/tr/watchlist.json';
import pumpsTr from './locales/tr/pumps.json';
import aiTr from './locales/tr/ai.json';
import alertsTr from './locales/tr/alerts.json';
import compareTr from './locales/tr/compare.json';
import signalsTr from './locales/tr/signals.json';

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
    ai: aiEn,
    alerts: alertsEn,
    compare: compareEn,
    signals: signalsEn,
  },
  tr: {
    common: commonTr,
    markets: marketsTr,
    detail: detailTr,
    watchlist: watchlistTr,
    pumps: pumpsTr,
    ai: aiTr,
    alerts: alertsTr,
    compare: compareTr,
    signals: signalsTr,
  },
} as const;

export type AppResources = (typeof resources)['en'];
