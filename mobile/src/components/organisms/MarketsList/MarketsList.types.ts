import type { ReactElement } from 'react';
import type { MarketRowViewModel } from '@/components/organisms/MarketRow';

export type MarketsListProps = {
  rows: MarketRowViewModel[];
  isLoading: boolean;
  isLoadingMore?: boolean;
  hasMore?: boolean;
  emptyMessage: string | null;
  errorMessage: string | null;
  onRetry: () => void;
  onPressRow: (symbol: string) => void;
  onLoadMore?: () => void;
  ListHeaderComponent?: ReactElement | null;
  refreshing?: boolean;
  onRefresh?: () => void;
  isWatched?: (symbol: string) => boolean;
  onStarPress?: (symbol: string) => void;
};
