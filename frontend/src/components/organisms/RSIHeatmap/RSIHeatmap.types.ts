export type RSIHeatmapRow = {
  rank?: number;
  symbol?: string;
  base?: string;
  lastPrice?: string;
  priceChangePercent?: string;
  quoteVolume?: string;
  marketCapCirculating?: number | null;
  rsi?: number | null;
  zone?: 'oversold' | 'neutral' | 'overbought' | string;
  error?: string;
};

export type RSIHeatmapData = {
  exchange?: string;
  quote?: string;
  interval?: string;
  period?: number;
  oversold?: number;
  overbought?: number;
  averageRsi?: number | null;
  oversoldCount?: number;
  neutralCount?: number;
  overboughtCount?: number;
  items?: RSIHeatmapRow[];
  stale?: boolean;
  note?: string;
};

export type RSIHeatmapProps = {
  data?: RSIHeatmapData;
  isLoading?: boolean;
  onOpen?: (exchange: string, symbol: string) => void;
};
