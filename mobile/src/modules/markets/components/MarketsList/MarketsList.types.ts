import type { ReactElement } from 'react';
import type { MarketRowViewModel } from '../../markets.types';

export type MarketsListProps = {
  rows: MarketRowViewModel[];
  isLoading: boolean;
  emptyMessage: string | null;
  errorMessage: string | null;
  onRetry: () => void;
  onPressRow: (symbol: string) => void;
  ListHeaderComponent?: ReactElement | null;
  ListFooterComponent?: ReactElement | null;
  refreshing?: boolean;
  onRefresh?: () => void;
};
