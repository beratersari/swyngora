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
  /** Optional event markers (e.g. pump scanner hits) */
  markers?: CandleChartMarker[];
  /** Full-height vertical event lines (delist announce / halt). */
  vertLines?: CandleChartVertLine[];
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
