import type { MarketRowViewModel } from '../../markets.types';

export type { MarketRowViewModel };

export type MarketsPageViewModel = {
  title: string;
  exchanges: string[];
  selectedExchange: string;
  onSelectExchange: (exchange: string) => void;
  exchangesLoading: boolean;

  search: string;
  onSearchChange: (q: string) => void;

  quote: string;
  quoteOptions: string[];
  onQuoteChange: (quote: string) => void;

  availableTags: string[];
  selectedTags: string[];
  onToggleTag: (tag: string) => void;
  onClearTags: () => void;

  sort: string;
  order: 'asc' | 'desc';
  sortOptions: { value: string; label: string }[];
  onSortChange: (sort: string) => void;
  onOrderChange: (order: 'asc' | 'desc') => void;

  rows: MarketRowViewModel[];
  total: number;
  offset: number;
  limit: number;
  onNextPage: () => void;
  onPrevPage: () => void;
  canNext: boolean;
  canPrev: boolean;

  isLoading: boolean;
  isRefreshing: boolean;
  isPollingPaused: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;
  lastUpdatedLabel: string | null;
  summaryLabel: string | null;

  onRetry: () => void;
  onRefresh: () => void;
  onPressRow: (symbol: string) => void;
};

export type MarketsPageProps = {
  viewModel?: MarketsPageViewModel;
};
