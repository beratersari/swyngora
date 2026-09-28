export type DetailChartToolbarProps = {
  intervals: string[];
  interval: string;
  intervalsLoading?: boolean;
  onIntervalChange: (interval: string) => void;
  onRefresh?: () => void;
  isFetching?: boolean;
  /** Absolute return % threshold for pump/dump markers (e.g. 5 = ±5%). */
  pumpThresholdPct?: number;
  onPumpThresholdChange?: (pct: number) => void;
  /** Show pump markers on the chart (default true when threshold control is present). */
  showPumpMarkers?: boolean;
  onShowPumpMarkersChange?: (show: boolean) => void;
  showSignalMarkers?: boolean;
  onShowSignalMarkersChange?: (show: boolean) => void;
  /** Aggressive buy−sell histogram under the candles. */
  showDelta?: boolean;
  onShowDeltaChange?: (show: boolean) => void;
  /** Anchored CVD lines under the candles. */
  showCvd?: boolean;
  onShowCvdChange?: (show: boolean) => void;
  /** Estimated liquidation prices on the candle scale. */
  showLiqLevels?: boolean;
  onShowLiqLevelsChange?: (show: boolean) => void;
};
