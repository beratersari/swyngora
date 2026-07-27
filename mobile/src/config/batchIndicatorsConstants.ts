/** Caps and defaults aligned with backend indicators_batch + MBIND-A. */

/** Service max after dedupe */
export const BATCH_MAX_SYMBOLS = 50;

export const DEFAULT_BATCH_INTERVAL = '1h';
export const DEFAULT_BATCH_RSI_PERIOD = 14;
export const DEFAULT_BATCH_EMA_PERIODS = '12,26';

/** Slower than spot/quote poll to avoid upstream storm */
export const BATCH_INDICATORS_POLL_MS = 45_000;

/** Favorites: max pairs considered for batch (then chunked per exchange ≤50) */
export const BATCH_FAVORITES_ENRICH_CAP = 50;

/** Markets: enrich at most this many visible symbols */
export const BATCH_MARKETS_ENRICH_CAP = 30;

export const BATCH_INDICATORS_DISCLAIMER =
  'RSI/EMA are informational only — not financial advice.';
