import type { ReactElement } from 'react';
import type { WatchlistRowViewModel } from '@/components/organisms/watchlist-row';

export type WatchlistListProps = {
  rows: WatchlistRowViewModel[];
  isLoading: boolean;
  emptyMessage: string | null;
  errorMessage: string | null;
  onRetry: () => void;
  onPressRow: (exchange: string, symbol: string) => void;
  onUnstar: (exchange: string, symbol: string) => void;
  ListHeaderComponent?: ReactElement | null;
  refreshing?: boolean;
  onRefresh?: () => void;
};
