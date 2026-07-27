import type { SpotSortField } from '@/libs/api';

export const DEFAULT_EXCHANGE = 'binance';
export const DEFAULT_QUOTE = 'USDT';
export const DEFAULT_SORT: SpotSortField = 'quoteVolume';
export const DEFAULT_ORDER = 'desc' as const;
export const DEFAULT_LIMIT = 30;
export const SPOT_POLL_MS = 10_000;
export const SEARCH_DEBOUNCE_MS = 300;

export const QUOTE_OPTIONS = ['USDT', 'USD', 'BTC', 'EUR'] as const;

export const SORT_OPTIONS: { value: SpotSortField; label: string }[] = [
  { value: 'quoteVolume', label: 'Quote vol' },
  { value: 'priceChangePercent', label: '24h %' },
  { value: 'lastPrice', label: 'Price' },
  { value: 'marketCapCirculating', label: 'Mcap' },
  { value: 'symbol', label: 'Symbol' },
];

export const FALLBACK_EXCHANGES = ['binance', 'coinbase', 'bybit'];
