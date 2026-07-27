export type WatchlistRowViewModel = {
  id: string;
  exchange: string;
  symbol: string;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  quoteLoading?: boolean;
};

export type WatchlistRowProps = {
  row: WatchlistRowViewModel;
  onPress?: (exchange: string, symbol: string) => void;
  onUnstar?: (exchange: string, symbol: string) => void;
};
