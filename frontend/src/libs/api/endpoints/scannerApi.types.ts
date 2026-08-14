/** Scanner DTOs — wire shapes from Go handler (OpenAPI ops lack named schemas). */

export type ScannerRuleType = 'rsi' | 'ma_crossover' | 'volume_increase';

export type ScannerRsiCondition = 'above' | 'below';

export type ScannerMaDirection = 'golden_cross' | 'death_cross';

export type ScannerRule = {
  id: string;
  clientId?: string;
  type: ScannerRuleType;
  interval: string;
  enabled: boolean;
  rsiPeriod?: number;
  rsiCondition?: ScannerRsiCondition;
  rsiThreshold?: number;
  maFastPeriod?: number;
  maSlowPeriod?: number;
  maDirection?: ScannerMaDirection;
  volumeLookback?: number;
  volumeMinRatio?: number;
  createdAt?: string;
  updatedAt?: string;
};

export type ScannerResult = {
  id: string;
  clientId?: string;
  ruleId: string;
  exchange: string;
  symbol: string;
  ruleType: ScannerRuleType;
  interval: string;
  marketDataKey: string;
  matchedAt: string;
  summary: string;
  metrics?: Record<string, number>;
};

export type ScannerBacktestStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'canceled'
  | 'failed';

export type ScannerBacktest = {
  id: string;
  clientId?: string;
  ruleId: string;
  exchange: string;
  symbol: string;
  interval: string;
  rangeStart: string;
  rangeEnd: string;
  status: ScannerBacktestStatus;
  progressPct: number;
  processedBars?: number;
  totalBars?: number;
  signalCount: number;
  errorMessage?: string;
  createdAt?: string;
  startedAt?: string | null;
  finishedAt?: string | null;
};

export type ScannerBacktestSignal = {
  id: string;
  backtestId: string;
  signalAt: string;
  closePrice: number;
  summary: string;
  return1d?: number;
  return5d?: number;
  return20d?: number;
  metrics?: Record<string, number>;
};

export type CreateScannerRuleArg = {
  type: ScannerRuleType;
  interval?: string;
  rsiPeriod?: number;
  rsiCondition?: ScannerRsiCondition;
  rsiThreshold?: number;
  maFastPeriod?: number;
  maSlowPeriod?: number;
  maDirection?: ScannerMaDirection;
  volumeLookback?: number;
  volumeMinRatio?: number;
};

export type StartScannerBacktestArg = {
  ruleId: string;
  symbol: string;
  exchange?: 'binance' | 'coinbase' | 'bybit' | 'nasdaq' | 'bist';
  rangeStart: string;
  rangeEnd: string;
};

export type ScannerRuleListResponse = {
  clientId?: string;
  count?: number;
  rules?: ScannerRule[];
};

export type ScannerResultListResponse = {
  clientId?: string;
  count?: number;
  total?: number;
  limit?: number;
  offset?: number;
  results?: ScannerResult[];
};

export type ScannerBacktestListResponse = {
  clientId?: string;
  count?: number;
  total?: number;
  backtests?: ScannerBacktest[];
};

export type ScannerBacktestSignalListResponse = {
  backtestId?: string;
  count?: number;
  total?: number;
  signalCount?: number;
  signals?: ScannerBacktestSignal[];
};
