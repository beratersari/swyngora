import type { SpotMarket, SpotSortField, SpotSortOrder } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';
import type { SpotMetricDef } from '@/libs/utils';

export type MarketsTableProps = WithLoadingProps & {
  items: SpotMarket[];
  exchange: string;
  sort: SpotSortField;
  order: SpotSortOrder;
  total: number;
  limit: number;
  offset: number;
  errorMessage?: string | null;
  onSortChange: (sort: SpotSortField, order: SpotSortOrder) => void;
  onPageChange: (offset: number, limit: number) => void;
  onRetry?: () => void;
  /** Navigate to coin detail when a row is activated */
  onRowOpen?: (symbol: string) => void;
  /** Set of `exchange:symbol` keys currently on the watchlist */
  watchedKeys?: Set<string>;
  /** Toggle watchlist for a pair (star control) */
  onToggleWatch?: (symbol: string, watched: boolean) => void;
  /**
   * Ordered metric columns to show (from shared catalog / column picker).
   * When omitted, defaults to the catalog's markets defaults.
   */
  metrics?: SpotMetricDef[];
};
