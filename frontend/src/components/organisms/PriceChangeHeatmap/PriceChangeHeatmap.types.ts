export type HeatmapMetric = 'quoteVolume' | 'marketCap';

export type PriceChangeHeatmapItem = {
  symbol: string;
  exchange: string;
  lastPrice?: string;
  priceChangePercent?: string;
  quoteVolume?: string;
  marketCapCirculating?: number | null;
};

export type HeatmapTile = {
  symbol: string;
  exchange: string;
  lastPrice?: string;
  quoteVolume?: string;
  marketCapCirculating?: number | null;
  changePct: number;
  weight: number;
  x: number;
  y: number;
  w: number;
  h: number;
};

export type PriceChangeHeatmapProps = {
  items: PriceChangeHeatmapItem[];
  metric?: HeatmapMetric;
  isLoading?: boolean;
  onOpen?: (exchange: string, symbol: string) => void;
};
