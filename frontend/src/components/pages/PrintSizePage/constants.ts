export const PRINT_SIZE_EXCHANGES = ['all', 'binance', 'bybit', 'okx'] as const;
export type PrintSizeExchange = (typeof PRINT_SIZE_EXCHANGES)[number];

export const PRINT_SIZE_WINDOWS = ['5m', '15m'] as const;
export type PrintSizeWindowId = (typeof PRINT_SIZE_WINDOWS)[number];
export const PRINT_SIZE_LIMIT = 20;
