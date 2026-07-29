import type { MarketRowViewModel } from '@/components/organisms/market-row';
import type { LeaderboardKind } from '@/config/leaderboardConstants';

export type LeaderboardsPageViewModel = {
  title: string;
  board: LeaderboardKind;
  boardOptions: { value: LeaderboardKind; label: string }[];
  onSelectBoard: (board: string) => void;

  exchanges: string[];
  selectedExchange: string;
  onSelectExchange: (exchange: string) => void;
  exchangesLoading: boolean;

  quote: string;
  quoteOptions: string[];
  onSelectQuote: (quote: string) => void;

  rows: MarketRowViewModel[];
  isLoading: boolean;
  isLoadingMore: boolean;
  isRefreshing: boolean;
  hasMore: boolean;
  isPollingPaused: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;
  summaryLabel: string | null;

  onLoadMore: () => void;
  onRetry: () => void;
  onRefresh: () => void;
  onPressRow: (symbol: string) => void;
  onBack: () => void;
  backLabel: string;
  retryLabel: string;
};

export type LeaderboardsPageProps = {
  viewModel?: LeaderboardsPageViewModel;
};
