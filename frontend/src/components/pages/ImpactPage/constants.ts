export const IMPACT_EXCHANGES = ['all', 'binance', 'coinbase', 'bybit'] as const;
export type ImpactExchange = (typeof IMPACT_EXCHANGES)[number];
export const IMPACT_SIDES = ['buy', 'sell'] as const;
export type ImpactSide = (typeof IMPACT_SIDES)[number];
export const IMPACT_NOTIONALS = [10_000, 100_000, 1_000_000] as const;
export const IMPACT_LIMIT = 20;
