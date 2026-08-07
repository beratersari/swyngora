import type {
  MarketExchange,
  ScannerBacktest,
  ScannerBacktestSignal,
  ScannerRule,
} from '@/libs/api';

export type SignalsBacktestRangeOption = { value: string; label: string };

export type SignalsBacktestPanelProps = {
  rules: ScannerRule[];
  jobs: ScannerBacktest[];
  signals: ScannerBacktestSignal[];
  rangeOptions: SignalsBacktestRangeOption[];
  selectedId?: string | null;
  selectedJob?: ScannerBacktest | null;
  loading?: boolean;
  signalsLoading?: boolean;
  startLoading?: boolean;
  cancelLoading?: boolean;
  startError?: unknown;
  onStart: (input: {
    ruleId: string;
    symbol: string;
    exchange: MarketExchange;
    rangeKey: string;
  }) => void;
  onSelect: (id: string) => void;
  onCancel: (id: string) => void;
};
