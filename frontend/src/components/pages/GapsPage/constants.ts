export const GAP_EXCHANGES = ['all', 'binance', 'coinbase', 'bybit'] as const;
export type GapExchange = (typeof GAP_EXCHANGES)[number];
export const GAP_SCAN_LIMIT = 20;
