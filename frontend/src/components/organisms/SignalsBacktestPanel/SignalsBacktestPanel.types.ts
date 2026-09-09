import type {
  CustomSignalProfile,
  MarketExchange,
  ScannerBacktest,
  ScannerBacktestPerformance,
  ScannerBacktestSignal,
  ScannerRule,
} from '@/libs/api';

export type SignalsBacktestRangeOption = { value: string; label: string };

export type SignalsBacktestSource = 'rule' | 'profile';

export type SignalsBacktestPanelProps = {
  rules: ScannerRule[];
  profiles: CustomSignalProfile[];
  jobs: ScannerBacktest[];
  signals: ScannerBacktestSignal[];
  performance?: ScannerBacktestPerformance | null;
  rangeOptions: SignalsBacktestRangeOption[];
  selectedId?: string | null;
  selectedJob?: ScannerBacktest | null;
  loading?: boolean;
  signalsLoading?: boolean;
  startLoading?: boolean;
  cancelLoading?: boolean;
  startError?: unknown;
  onStart: (input: {
    ruleId?: string;
    profileId?: string;
    symbol: string;
    exchange: MarketExchange;
    rangeKey: string;
  }) => void;
  onSelect: (id: string) => void;
  onCancel: (id: string) => void;
};
