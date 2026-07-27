export type MarketRowViewModel = {
  id: string;
  symbol: string;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  quoteVolumeLabel: string;
  marketCapLabel: string;
  tagsLabel: string;
};

export type MarketRowProps = {
  row: MarketRowViewModel;
  onPress?: (symbol: string) => void;
  /** When set, shows star control */
  watched?: boolean;
  onStarPress?: (symbol: string) => void;
};
