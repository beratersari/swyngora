/** Default UI poll interval for spot list (ms). Backend spot TTL ~5s. */
export const DEFAULT_SPOT_POLL_MS = 10_000;

/** Detail ticker / supply refresh while tab is visible (ms). Backend ticker TTL ~15s. */
export const DEFAULT_DETAIL_TICKER_POLL_MS = 15_000;

/** Candles + indicators refresh while tab is visible (ms). Backend candles TTL ~30s. */
export const DEFAULT_DETAIL_SERIES_POLL_MS = 30_000;

/** Default candle / indicator bar count on coin detail. */
export const DEFAULT_DETAIL_CANDLE_LIMIT = 100;

/** Default RSI period (matches backend). */
export const DEFAULT_RSI_PERIOD = 14;

/** Default EMA periods (matches backend). */
export const DEFAULT_EMA_PERIODS = '12,26';

/** Preferred default interval when venue supports it. */
export const DEFAULT_DETAIL_INTERVAL = '1h';

export const APP_NAME = 'Swyngora';

export const SUPPORTED_EXCHANGES = ['binance', 'coinbase', 'bybit'] as const;

export type SupportedExchange = (typeof SUPPORTED_EXCHANGES)[number];
