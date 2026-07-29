import type { AppLocale, I18nNamespace } from './config';
import { I18N_NAMESPACES, SUPPORTED_LOCALES } from './config';

import commonEn from './locales/en/common.json';
import homeEn from './locales/en/home.json';
import marketsEn from './locales/en/markets.json';
import watchlistEn from './locales/en/watchlist.json';
import pumpsEn from './locales/en/pumps.json';
import detailEn from './locales/en/detail.json';
import aiEn from './locales/en/ai.json';
import leaderboardsEn from './locales/en/leaderboards.json';

import commonTr from './locales/tr/common.json';
import homeTr from './locales/tr/home.json';
import marketsTr from './locales/tr/markets.json';
import watchlistTr from './locales/tr/watchlist.json';
import pumpsTr from './locales/tr/pumps.json';
import detailTr from './locales/tr/detail.json';
import aiTr from './locales/tr/ai.json';
import leaderboardsTr from './locales/tr/leaderboards.json';

/**
 * Bundled translation catalogs.
 *
 * To add a language:
 * 1. Create `locales/<code>/{common,home,markets,watchlist,pumps,detail,ai,leaderboards}.json`
 * 2. Import and register in `localeBundles` below
 * 3. Add `<code>` to `SUPPORTED_LOCALES` in `config.ts`
 * 4. Add display labels under `common.language.<code>` and `LOCALE_META`
 *
 * To add a namespace: add JSON files + import + extend `I18N_NAMESPACES`.
 */
const localeBundles = {
  en: {
    common: commonEn,
    home: homeEn,
    markets: marketsEn,
    watchlist: watchlistEn,
    pumps: pumpsEn,
    detail: detailEn,
    ai: aiEn,
    leaderboards: leaderboardsEn,
  },
  tr: {
    common: commonTr,
    home: homeTr,
    markets: marketsTr,
    watchlist: watchlistTr,
    pumps: pumpsTr,
    detail: detailTr,
    ai: aiTr,
    leaderboards: leaderboardsTr,
  },
} as const satisfies Record<AppLocale, Record<I18nNamespace, object>>;

export type AppResources = (typeof localeBundles)['en'];

/** i18next resources map built from the registry (keeps init flexible). */
export const resources: Record<string, AppResources> = Object.fromEntries(
  SUPPORTED_LOCALES.map((lng) => [lng, localeBundles[lng]]),
);

/** Runtime guard used by tests / future dynamic loaders. */
export function hasLocaleBundle(locale: string): locale is AppLocale {
  return locale in localeBundles;
}

export function listRegisteredNamespaces(): readonly I18nNamespace[] {
  return I18N_NAMESPACES;
}
