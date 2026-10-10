export const DERIVATIVES_EXCHANGES = ['all', 'binance', 'bybit', 'okx', 'hyperliquid'] as const;
export type DerivativesExchange = (typeof DERIVATIVES_EXCHANGES)[number];

export const DERIVATIVES_LIMITS = [10, 20, 36, 100] as const;
