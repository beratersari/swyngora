export type IntervalToolbarProps = {
  intervals: string[];
  selected: string;
  onSelect: (interval: string) => void;
  isLoading?: boolean;
  showEma: boolean;
  onToggleEma: () => void;
  /** Pump/dump markers on the OHLCV chart. */
  showPumps?: boolean;
  onTogglePumps?: () => void;
  /** High/low margin price lines for pump events. */
  showPumpMargin?: boolean;
  onTogglePumpMargin?: () => void;
};
