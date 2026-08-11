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
};
