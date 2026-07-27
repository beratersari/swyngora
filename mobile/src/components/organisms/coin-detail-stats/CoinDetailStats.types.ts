export type CoinDetailStatsProps = {
  items: { label: string; value: string }[];
  isLoading?: boolean;
  tickerError?: string | null;
  supplyError?: string | null;
};
