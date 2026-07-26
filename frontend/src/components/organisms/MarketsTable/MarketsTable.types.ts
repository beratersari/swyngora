import type { SpotMarket, SpotSortField, SpotSortOrder } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

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
};
