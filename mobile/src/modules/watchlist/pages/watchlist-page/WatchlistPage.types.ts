import type { WatchlistRowViewModel } from '@/components/organisms/watchlist-row';

export type WatchlistPageViewModel = {
  title: string;
  countLabel: string | null;
  isLoading: boolean;
  isRefreshing: boolean;
  isPollingPaused: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;
  actionError: string | null;
  /** Pairs to enrich (capped); page maps to quote-connected rows */
  pairs: { exchange: string; symbol: string }[];
  onRetry: () => void;
  onRefresh: () => void;
  onPressRow: (exchange: string, symbol: string) => void;
  onUnstar: (exchange: string, symbol: string) => void;
  onOpenMarkets: () => void;
  pollQuotes: boolean;
};

export type WatchlistPageProps = {
  viewModel?: WatchlistPageViewModel;
};

export type { WatchlistRowViewModel };
