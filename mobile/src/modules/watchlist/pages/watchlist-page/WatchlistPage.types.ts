import type { WatchlistRowViewModel } from '@/components/organisms/watchlist-row';

import type { RsiRowFields } from '@/libs/utils';

export type WatchlistPageViewModel = {
  title: string;
  countLabel: string | null;
  isLoading: boolean;
  isRefreshing: boolean;
  isPollingPaused: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;
  actionError: string | null;
  /** Batch indicators banner (whole-request failure) */
  indicatorsError: string | null;
  /** Informational note under list chrome */
  indicatorsDisclaimer: string | null;
  /** Pairs to enrich (capped); page maps to quote-connected rows */
  pairs: { exchange: string; symbol: string }[];
  onRetry: () => void;
  onRefresh: () => void;
  onPressRow: (exchange: string, symbol: string) => void;
  onUnstar: (exchange: string, symbol: string) => void;
  onOpenMarkets: () => void;
  pollQuotes: boolean;
  /** Batch RSI map key: exchange|SYMBOL */
  rsiByKey: Map<string, RsiRowFields>;
};

export type WatchlistPageProps = {
  viewModel?: WatchlistPageViewModel;
};

export type { WatchlistRowViewModel };
