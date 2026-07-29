import type { MarketExchange } from '@/libs/api';

export type LeaderboardKind = 'gainers' | 'losers' | 'volume';

export const LEADERBOARD_KINDS: readonly LeaderboardKind[] = [
  'gainers',
  'losers',
  'volume',
] as const;

export const LEADERBOARD_DEFAULT_EXCHANGE: MarketExchange = 'binance';
export const LEADERBOARD_DEFAULT_QUOTE = 'USDT';
export const LEADERBOARD_PAGE_SIZE = 30;
export const LEADERBOARD_POLL_MS = 15_000;

export const LEADERBOARD_QUOTE_OPTIONS = ['USDT', 'USD', 'USDC', 'BTC'] as const;

export const FALLBACK_LEADERBOARD_EXCHANGES = ['binance', 'coinbase', 'bybit'] as const;

export function isLeaderboardKind(value: string | undefined | null): value is LeaderboardKind {
  return (
    value === 'gainers' || value === 'losers' || value === 'volume'
  );
}

/** Default quote for an exchange (Coinbase prefers USD). */
export function defaultQuoteForLeaderboard(exchange: string): string {
  return String(exchange).toLowerCase() === 'coinbase' ? 'USD' : LEADERBOARD_DEFAULT_QUOTE;
}
