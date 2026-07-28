export type PumpScanFiltersProps = {
  exchanges: string[];
  selectedExchange: string;
  onSelectExchange: (exchange: string) => void;
  exchangesLoading?: boolean;

  lookbackHours: number;
  lookbackOptions: number[];
  onSelectLookback: (hours: number) => void;

  minReturnPct: number;
  thresholdOptions: number[];
  onSelectThreshold: (pct: number) => void;

  direction: string;
  directionOptions: { value: string; label: string }[];
  onSelectDirection: (direction: string) => void;

  summaryLabel?: string | null;
};
