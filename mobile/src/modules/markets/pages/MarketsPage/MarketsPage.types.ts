import type { MarketRowViewModel } from '@/components/organisms/MarketRow';

export type { MarketRowViewModel };

export type MarketsPageViewModel = {
  title: string;
  exchanges: string[];
  selectedExchange: string;
  onSelectExchange: (exchange: string) => void;
  exchangesLoading: boolean;

  search: string;
  onSearchChange: (q: string) => void;
  isSearchDebouncing: boolean;

  activeFilterCount: number;
  filterSummary: string | null;
  onOpenFilters: () => void;

  favoritesOnly: boolean;
  favoritesCount: number;
  onToggleFavoritesOnly: () => void;

  rows: MarketRowViewModel[];
  total: number;
  hasMore: boolean;
  isLoading: boolean;
  isLoadingMore: boolean;
  isRefreshing: boolean;
  isPollingPaused: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;
  summaryLabel: string | null;
  detailHint: string | null;
  actionError: string | null;

  onLoadMore: () => void;
  onRetry: () => void;
  onRefresh: () => void;
  onPressRow: (symbol: string) => void;
  isWatched: (symbol: string) => boolean;
  onStarPress: (symbol: string) => void;
};

export type MarketsPageProps = {
  viewModel?: MarketsPageViewModel;
};
