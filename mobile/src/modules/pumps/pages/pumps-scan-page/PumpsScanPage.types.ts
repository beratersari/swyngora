import type { PumpHitRowViewModel } from '@/components/organisms/pump-hit-row';

export type PumpsScanPageViewModel = {
  title: string;
  exchanges: string[];
  selectedExchange: string;
  onSelectExchange: (exchange: string) => void;
  exchangesLoading: boolean;

  lookbackHours: number;
  lookbackOptions: number[];
  onSelectLookback: (hours: number) => void;

  minReturnPct: number;
  thresholdOptions: number[];
  onSelectThreshold: (pct: number) => void;

  direction: string;
  directionOptions: { value: string; label: string }[];
  onSelectDirection: (direction: string) => void;

  summaryLabel: string | null;
  disclaimer: string;

  rows: PumpHitRowViewModel[];
  isLoading: boolean;
  isRefreshing: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;

  onRetry: () => void;
  onRefresh: () => void;
  onPressRow: (exchange: string, symbol: string) => void;
};

export type PumpsScanPageProps = {
  viewModel?: PumpsScanPageViewModel;
};
