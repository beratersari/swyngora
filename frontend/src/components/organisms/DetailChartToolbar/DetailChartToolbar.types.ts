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
  /** Resting bid and ask size behind the candles. */
  showBook?: boolean;
  onShowBookChange?: (show: boolean) => void;
  /** Buy versus sell at each price for the visible window. */
  showFootprint?: boolean;
  onShowFootprintChange?: (show: boolean) => void;
  /** Binance crowd long percent versus top-trader long percent. */
  showCrowd?: boolean;
  onShowCrowdChange?: (show: boolean) => void;
  /** Stored USDT open interest for Binance, Bybit, and OKX. */
  showOi?: boolean;
  onShowOiChange?: (show: boolean) => void;
  /** Stored funding percent for Binance, Bybit, and OKX. */
  showFunding?: boolean;
  onShowFundingChange?: (show: boolean) => void;
  /** Short liquidation notional minus long liquidation notional in each candle. */
  showLiqNotional?: boolean;
  onShowLiqNotionalChange?: (show: boolean) => void;
  /** Price × time liquidation intensity behind the candles. */
  showLiqMap?: boolean;
  onShowLiqMapChange?: (show: boolean) => void;
  /** Bands that stay until a later candle trades through them. */
  showOpenLevels?: boolean;
  onShowOpenLevelsChange?: (show: boolean) => void;
  liqMapRange?: '12h' | '24h' | '3d' | '7d' | '14d' | '30d' | '90d';
  onLiqMapRangeChange?: (range: '12h' | '24h' | '3d' | '7d' | '14d' | '30d' | '90d') => void;
  liqMapVenue?: 'combined' | 'binance' | 'bybit' | 'okx';
  onLiqMapVenueChange?: (venue: 'combined' | 'binance' | 'bybit' | 'okx') => void;
  liqMapSide?: 'totals' | 'longs' | 'shorts';
  onLiqMapSideChange?: (side: 'totals' | 'longs' | 'shorts') => void;
};
