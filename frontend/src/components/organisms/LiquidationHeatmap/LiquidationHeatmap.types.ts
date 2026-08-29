import type {
  LiquidationHuntHeatmap,
  LiquidationHuntHeatmapGrid,
  LiquidationHuntHeatmapReviewVenue,
} from '@/libs/api/endpoints/marketApi.types';

export type { LiquidationHuntHeatmapReviewVenue };

export type LiqHeatRange = '12h' | '24h' | '3d' | '7d';
export type LiqHeatVenue = 'combined' | 'binance' | 'bybit';
export type LiqHeatSide = 'totals' | 'longs' | 'shorts';

export type LiquidationHeatmapData = LiquidationHuntHeatmap;
export type LiquidationHeatmapGrid = LiquidationHuntHeatmapGrid;

export type LiqHeatHover = {
  x: number;
  y: number;
  timeMs: number;
  price: number;
  longs: number;
  shorts: number;
  totals: number;
};

export type LiqHeatLayout = {
  plotX: number;
  plotY: number;
  plotW: number;
  plotH: number;
  cellW: number;
  cellH: number;
  nT: number;
  nP: number;
  times: number[];
  prices: number[];
};

export type LiquidationHeatmapProps = {
  data?: LiquidationHeatmapData;
  range: LiqHeatRange;
  onRangeChange: (range: LiqHeatRange) => void;
  venue: LiqHeatVenue;
  onVenueChange: (venue: LiqHeatVenue) => void;
  side: LiqHeatSide;
  onSideChange: (side: LiqHeatSide) => void;
  lastPrice?: number;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
};
