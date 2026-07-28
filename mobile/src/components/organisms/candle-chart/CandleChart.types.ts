import type { ChartCandle, ChartLinePoint, ChartMarker, ChartPriceLine } from '@/libs/utils';

export type CandleChartOverlay = {
  id: string;
  title: string;
  color: string;
  data: ChartLinePoint[];
};

export type { ChartMarker, ChartPriceLine };

export type CandleChartProps = {
  candles: ChartCandle[];
  overlays?: CandleChartOverlay[];
  /** Pump/dump markers (series markers plugin). */
  markers?: ChartMarker[];
  /** Horizontal margin / level lines (high-low / start-end of pumps). */
  priceLines?: ChartPriceLine[];
  height?: number;
  isLoading?: boolean;
  /** Fetching bars to the left of the series (pan back in time). */
  isLoadingOlder?: boolean;
  errorMessage?: string | null;
  emptyMessage?: string;
  /**
   * Identity of the series (exchange|symbol|interval). Resets fit/scroll when it changes.
   */
  seriesKey?: string;
  /** Fired when the user pans near the oldest loaded bar. */
  onRequestOlderHistory?: () => void;
  /** When false, chart will not request more history. Default true. */
  canLoadOlder?: boolean;
  /** Bars remaining to the left of the viewport before requesting more history. */
  historyEdgeBars?: number;
};
