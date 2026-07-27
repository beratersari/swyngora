export const DEFAULT_DETAIL_INTERVAL = '1h';
/** Initial + polled latest window. */
export const DEFAULT_DETAIL_CANDLE_LIMIT = 200;
/** Bars to request when user pans left for more history. */
export const DETAIL_HISTORY_PAGE_SIZE = 200;
/**
 * Request older history when fewer than this many bars sit before the
 * left edge of the visible range (Lightweight Charts barsBefore).
 */
export const DETAIL_HISTORY_EDGE_BARS = 20;
/** Cap merged series (API max per request is 1000). */
export const DETAIL_MAX_CANDLES = 1000;
export const DEFAULT_RSI_PERIOD = 14;
export const DEFAULT_EMA_PERIODS = '12,26';
export const DETAIL_TICKER_POLL_MS = 15_000;
export const DETAIL_SERIES_POLL_MS = 30_000;
