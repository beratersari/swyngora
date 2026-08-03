export type DashboardMarketRowData = {
  id: string;
  exchange: string;
  symbol: string;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  metaLabel?: string;
};

export type DashboardMarketRowProps = {
  row: DashboardMarketRowData;
  onPress?: (exchange: string, symbol: string) => void;
};
