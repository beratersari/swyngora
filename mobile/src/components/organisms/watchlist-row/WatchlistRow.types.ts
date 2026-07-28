export type WatchlistRowViewModel = {
  id: string;
  exchange: string;
  symbol: string;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  quoteLoading?: boolean;
  /** Optional batch RSI enrichment */
  rsiLabel?: string;
  rsiTone?: 'success' | 'warning' | 'error' | 'secondary';
  rsiLoading?: boolean;
};

export type WatchlistRowProps = {
  row: WatchlistRowViewModel;
  onPress?: (exchange: string, symbol: string) => void;
  onUnstar?: (exchange: string, symbol: string) => void;
};
