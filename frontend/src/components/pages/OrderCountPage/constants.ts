export const ORDER_COUNT_EXCHANGES = ['all', 'binance', 'bybit', 'okx'] as const;
export type OrderCountExchange = (typeof ORDER_COUNT_EXCHANGES)[number];

export const ORDER_COUNT_WINDOWS = ['1m', '5m', '15m'] as const;
export type OrderCountWindowId = (typeof ORDER_COUNT_WINDOWS)[number];
export const ORDER_COUNT_LIMIT = 20;
