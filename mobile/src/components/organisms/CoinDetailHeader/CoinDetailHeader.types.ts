export type CoinDetailHeaderProps = {
  symbol: string;
  exchange: string;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  isLoading?: boolean;
  onBack: () => void;
  watched?: boolean;
  onStarPress?: () => void;
};
