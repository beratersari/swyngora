import type {
  LiquidationHuntHeatmap,
  LiquidationHuntHeatmapGrid,
  LiquidationHuntHeatmapReviewSignal,
  LiquidationHuntHeatmapReviewSignalHorizon,
  LiquidationHuntHeatmapReviewVenue,
} from '@/libs/api/endpoints/marketApi.types';

export type {
  LiquidationHuntHeatmapReviewSignal,
  LiquidationHuntHeatmapReviewSignalHorizon,
  LiquidationHuntHeatmapReviewVenue,
};

export type LiqHeatRange = '12h' | '24h' | '3d' | '7d' | '14d' | '30d' | '90d';
export type LiqHeatVenue = 'combined' | 'binance' | 'bybit' | 'okx';
export type LiqHeatSide = 'totals' | 'longs' | 'shorts';
export type LiqHeatSource = 'prints' | 'model';

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
  /** True when this print column was not recorded. A quiet recorded hour stays false. */
  unrecorded: boolean;
  /** Venues that lack this column. Combined names each one. */
  gapVenues: string[];
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
  /** Measured prints or the labeled leverage model. Chips render only when the callback is set. */
  source?: LiqHeatSource;
  onSourceChange?: (source: LiqHeatSource) => void;
  lastPrice?: number;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
};
