import type { MarketExchange } from '@/libs/api';

/** Stable venue order for compare rows. */
export const CROSS_EXCHANGE_VENUES: readonly MarketExchange[] = [
  'binance',
  'coinbase',
  'bybit',
] as const;

/** Max symbol candidates to try per non-source venue. */
export const CROSS_EXCHANGE_MAX_CANDIDATES = 3;
