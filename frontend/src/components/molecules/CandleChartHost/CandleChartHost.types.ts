import type { ChartCandle, ChartLinePoint } from '@/libs/utils';
import type { WithLoadingProps } from '@/components/types';
import type { CandleChartVertLine } from './CandleChartHost.vertLines';

export type { CandleChartVertLine };

export type CandleChartOverlay = {
  id: string;
  data: ChartLinePoint[];
  color: string;
  title?: string;
};

/** One series in a pane under the candles. Pane 1 is the first lower pane. */
export type CandleChartLowerSeries = {
  id: string;
  pane: number;
  kind: 'line' | 'histogram';
  data: ChartLinePoint[];
  color: string;
  /** Histogram bars below zero. Line series ignore this. */
  downColor?: string;
  title?: string;
};

/** One resting-size column drawn behind the candles. Time is unix seconds. */
export type CandleChartHeatColumn = {
  time: number;
  bids: { price: number; notional: number }[];
  asks: { price: number; notional: number }[];
};

/** Buy and sell size at one price for the visible window. */
export type CandleChartFootprintLevel = {
  price: number;
  buy: number;
  sell: number;
};

/** Estimated liquidation notional at one price. Wider bars are larger pockets. */
export type CandleChartLiqBar = {
  price: number;
  notional: number;
  side: 'long' | 'short';
};

/** Horizontal price on the candle scale (estimated liquidation levels). */
export type CandleChartPriceLine = {
  id: string;
  price: number;
  color: string;
  title: string;
};

/** Chart markers (pump/dump events) aligned to bar times (UTC seconds). */
export type CandleChartMarker = {
  time: number;
  position: 'aboveBar' | 'belowBar' | 'inBar';
  color: string;
  shape: 'arrowUp' | 'arrowDown' | 'circle' | 'square';
  text?: string;
};

export type CandleChartHostProps = WithLoadingProps & {
  data: ChartCandle[];
  /** Optional EMA (or other) line overlays on the price scale */
  overlays?: CandleChartOverlay[];
  /** CVD and buy−sell delta in panes under the candles. Same time scale. */
  lowerSeries?: CandleChartLowerSeries[];
  /** Dashed prices on the candle scale, such as estimated liquidation levels. */
  priceLines?: CandleChartPriceLine[];
  /** Resting bid and ask size drawn behind the candles. */
  bookHeat?: CandleChartHeatColumn[];
  /** Buy versus sell at each price, drawn on the right of the candle pane. */
  footprint?: CandleChartFootprintLevel[];
  /** Estimated liquidation pockets, drawn beside the footprint. */
  liqProfile?: CandleChartLiqBar[];
  /** Visible candle span in unix seconds, so the footprint window can follow the pan. */
  onVisibleSpan?: (fromSec: number, toSec: number) => void;
  /** Optional event markers (e.g. pump scanner hits) */
  markers?: CandleChartMarker[];
  /** Full-height vertical event lines (delist announce / halt). */
  vertLines?: CandleChartVertLine[];
  /** Seconds per bar so delist lines snap to the containing candle, not the nearest open. */
  barDurationSec?: number;
  height?: number;
  className?: string;
  /**
   * Resets fit/scroll bookkeeping when exchange, symbol, or interval changes
   * so a new series fits content instead of preserving the previous range.
   */
  seriesKey?: string;
  /** True while a progressive history fetch is in flight (suppresses repeat requests). */
  isLoadingMore?: boolean;
  /** False when the API has no older candles to offer (or max limit reached). */
  hasMoreHistory?: boolean;
  /**
   * Fired when the user scrolls/pans so the left edge of the visible range
   * approaches the oldest loaded bar — parent should increase candle limit.
   */
  onNeedMoreHistory?: () => void;
  rightPadding?: number;
  /** Index after the last real (non-carried) bar; initial view stays on last trade. */
  anchorEndIndex?: number;
};
