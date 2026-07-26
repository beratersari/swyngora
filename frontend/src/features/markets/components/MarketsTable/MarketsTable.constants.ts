import type { SpotSortField } from '@/libs/api';

export const PAGE_SIZE_OPTIONS = ['25', '50', '100'] as const;

/** Map Ant Design column key → API sort field */
export const COLUMN_SORT: Record<string, SpotSortField> = {
  symbol: 'symbol',
  lastPrice: 'lastPrice',
  priceChangePercent: 'priceChangePercent',
  quoteVolume: 'quoteVolume',
  marketCapCirculating: 'marketCapCirculating',
  tradeCount: 'tradeCount',
  tags: 'tags',
};
