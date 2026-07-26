/** Default UI poll interval for spot list (ms). Backend spot TTL ~5s. */
export const DEFAULT_SPOT_POLL_MS = 10_000;

export const APP_NAME = 'Swyngora';

export const SUPPORTED_EXCHANGES = ['binance', 'coinbase', 'bybit'] as const;

export type SupportedExchange = (typeof SUPPORTED_EXCHANGES)[number];
