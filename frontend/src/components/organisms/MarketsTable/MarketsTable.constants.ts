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

/**
 * Ant Design cycles with `sortDirections[indexOf(current) + 1]`.
 * With only `['ascend','descend']`, the step after `descend` is `undefined`,
 * which clears columnKey/field in onChange and breaks further clicks.
 * Repeating `ascend` at the end makes descend → ascend forever.
 * @see antd `useSorter` nextSortDirection
 */
export const SORT_DIRECTIONS = ['ascend', 'descend', 'ascend'] as const;
