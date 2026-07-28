/** localStorage / AsyncStorage key for opaque client id */
export const CLIENT_ID_STORAGE_KEY = 'swyngora.mobile.clientId.v1';

/** localStorage key for offline-friendly watchlist cache */
export const WATCHLIST_STORAGE_KEY = 'swyngora.mobile.watchlist.v1';

/** Mirrors backend domain.MaxWatchlistItems */
export const MAX_WATCHLIST_ITEMS = 200;

/** Prefix for generated client ids (never empty / "default") */
export const CLIENT_ID_PREFIX = 'mobile-';

/** Quote poll while Watchlist tab is focused + app active */
export const WATCHLIST_QUOTE_POLL_MS = 15_000;

/** Max concurrent quote enrichments (design §7) */
export const WATCHLIST_QUOTE_ENRICH_CAP = 40;
