export const SPREAD_EXCHANGES = ['all', 'binance', 'coinbase', 'bybit'] as const;
export type SpreadExchange = (typeof SPREAD_EXCHANGES)[number];
export const SPREAD_SCAN_LIMIT = 20;
