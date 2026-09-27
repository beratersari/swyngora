export const WALL_EXCHANGES = ['all', 'binance', 'coinbase', 'bybit'] as const;
export type WallExchange = (typeof WALL_EXCHANGES)[number];
export const WALL_SCAN_LIMIT = 40;
