export const UNUSUAL_EXCHANGES = ['all', 'binance', 'bybit', 'okx'] as const;
export const UNUSUAL_METRICS = ['all', 'funding', 'open_interest', 'liquidations', 'cvd', 'spread'] as const;
export const UNUSUAL_LIMITS = [10, 20, 36, 100] as const;
export const UNUSUAL_POLL_MS = 60_000;

export type UnusualExchange = (typeof UNUSUAL_EXCHANGES)[number];
export type UnusualMetric = (typeof UNUSUAL_METRICS)[number];
