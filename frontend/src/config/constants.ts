/** Default UI poll interval for spot list (ms). Backend spot TTL ~5s. */
export const DEFAULT_SPOT_POLL_MS = 10_000;

/** Detail ticker / supply refresh while tab is visible (ms). Backend ticker TTL ~15s. */
export const DEFAULT_DETAIL_TICKER_POLL_MS = 15_000;

/** Candles + indicators refresh while tab is visible (ms). Backend candles TTL ~30s. */
export const DEFAULT_DETAIL_SERIES_POLL_MS = 30_000;

/** Initial candle / indicator bar count on coin detail (grows as user pans left). */
export const DEFAULT_DETAIL_CANDLE_LIMIT = 100;

/** Extra candles loaded each time the chart approaches older history. */
export const DETAIL_CANDLE_PAGE_SIZE = 100;

/** Cap for progressive candle loads (matches OpenAPI getCandles max). */
export const DETAIL_CANDLE_MAX_LIMIT = 1000;

/** Default |return %| threshold for pump markers on the detail chart. */
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
