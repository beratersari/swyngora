/** Default UI poll interval for spot list (ms). Matches backend SPOT_MARKET_CACHE_TTL default (5s). */
export const DEFAULT_SPOT_POLL_MS = 5_000;

/** Detail ticker / supply refresh while tab is visible (ms). Backend ticker TTL ~15s. */
export const DEFAULT_DETAIL_TICKER_POLL_MS = 15_000;

/** Candles + indicators refresh while tab is visible (ms). Backend candles TTL ~30s. */
export const DEFAULT_DETAIL_SERIES_POLL_MS = 30_000;

/**
 * Live (polled) candle window on coin detail.
 * Sized so pump/dump markers have a useful first paint without requiring pan.
 * Deeper history still loads via endTime pages on pan-left.
 */
export const DEFAULT_DETAIL_CANDLE_LIMIT = 300;

/** Candles fetched per pan-left history page (API max per request is 1000). */
export const DETAIL_CANDLE_PAGE_SIZE = 200;

/**
 * Max bars kept in the chart client-side (live + history pages).
 * API still returns at most 1000 per request; we page with endTime for deeper history.
 * ~10k × 1h ≈ 14 months; ×15m ≈ 3.5 months; ×1d ≈ 27 years of daily.
 */
export const DETAIL_CANDLE_MAX_LIMIT = 10_000;

/**
 * Cap for indicator series and a floor for pump analysis (API max 1000).
 * Pump limit tracks loaded chart bars up to 1000 (see analyticsBarLimit).
 */
export const DETAIL_INDICATOR_LIMIT = 500;

/** Backend getCandles / pumps / indicators max `limit` per request. */
export const DETAIL_API_BAR_MAX = 1000;/** Default |return %| threshold for pump markers on the detail chart. */
export const DEFAULT_DETAIL_PUMP_THRESHOLD_PCT = 5;

/** Preset pump thresholds offered in the chart toolbar (%). */
export const DETAIL_PUMP_THRESHOLD_OPTIONS = [3, 5, 8, 10, 15, 20] as const;

/** Default RSI period (matches backend). */
export const DEFAULT_RSI_PERIOD = 14;

/** Default EMA periods (matches backend). */
export const DEFAULT_EMA_PERIODS = '12,26';

/** Preferred default interval when venue supports it. */
export const DEFAULT_DETAIL_INTERVAL = '1h';

export const APP_NAME = 'Swyngora';

export const SUPPORTED_EXCHANGES = ['binance', 'coinbase', 'bybit'] as const;

export type SupportedExchange = (typeof SUPPORTED_EXCHANGES)[number];
