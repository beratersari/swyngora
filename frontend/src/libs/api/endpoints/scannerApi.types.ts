/** Scanner DTOs — wire shapes from Go handler (OpenAPI ops lack named schemas). */

export type ScannerCondition = 'rsi' | 'ma_crossover' | 'volume_increase';

export type ScannerRuleType = ScannerCondition | 'combo';

export type ScannerMatchMode = 'all' | 'any';

export type ScannerRsiCondition = 'above' | 'below';

export type ScannerMaDirection = 'golden_cross' | 'death_cross';

export type ScannerRule = {
  id: string;
  clientId?: string;
  type: ScannerRuleType;
  conditions?: ScannerCondition[];
  matchMode?: ScannerMatchMode;
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

export type ScannerBacktestScope = 'symbol' | 'watchlist';

export type ScannerBacktestMarket = {
  exchange: string;
  symbol: string;
};

export type ScannerBacktest = {
  id: string;
  clientId?: string;
  ruleId: string;
  kind?: 'scanner_rule' | 'custom_profile';
  sourceName?: string;
  exchange: string;
  symbol: string;
  scope?: ScannerBacktestScope;
  markets?: ScannerBacktestMarket[];
  interval: string;
  rangeStart: string;
  rangeEnd: string;
  status: ScannerBacktestStatus;
  progressPct: number;
  processedBars?: number;
  totalBars?: number;
  signalCount: number;
  skippedBars?: number;
  missingFactors?: string[];
  errorMessage?: string;
  createdAt?: string;
  startedAt?: string | null;
  finishedAt?: string | null;
  performance?: ScannerBacktestPerformance;
};

export type ScannerBacktestSignalFactor = {
  id: string;
  label: string;
  score: number;
  weightPct: number;
  sharePct?: number;
  effect?: number;
  status: string;
  detail?: string;
};

export type ScannerBacktestTradeOutcome = 'tp' | 'sl' | 'open' | 'tp1' | 'tp1_tp2' | 'tp1_sl' | 'tp1_open';

export type ScannerBacktestTrade = {
  side: 'long' | 'short';
  leverage: number;
  entry: number;
  stopLoss: number;
  takeProfit: number;
  takeProfit1?: number;
  takeProfit2?: number;
  useTp2?: boolean;
  tp1ClosePct?: number;
  stopAfterTp1?: 'keep' | 'entry' | 'trail';
  stopMovedToEntry?: boolean;
  trailStop?: number;
  feePct?: number;
  slipPct?: number;
  outcome: ScannerBacktestTradeOutcome;
  exitPrice?: number;
  pnlPct: number;
  barsHeld?: number;
  hitAt?: string;
  hitTimeUnknown?: boolean;
  closedKnownAt?: string;
  enteredAt?: string;
  tp1ExitPrice?: number;
  tp1PnlPct?: number;
  remainderPct?: number;
  marks?: { at: string; price: number }[];
};

export type ScannerBacktestSignal = {
  id: string;
  backtestId: string;
  exchange?: string;
  symbol?: string;
  signalAt: string;
  closePrice: number;
  summary: string;
  return1d?: number;
  return5d?: number;
  return20d?: number;
  metrics?: Record<string, number>;
  factors?: ScannerBacktestSignalFactor[];
  trade?: ScannerBacktestTrade;
};

export type CreateScannerRuleArg = {
  type?: ScannerRuleType;
  conditions?: ScannerCondition[];
  matchMode?: ScannerMatchMode;
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

export type UpdateScannerRuleArg = CreateScannerRuleArg & {
  id: string;
  enabled?: boolean;
};

export type StartScannerBacktestArg = {
  ruleId?: string;
  profileId?: string;
  symbol?: string;
  scope?: ScannerBacktestScope;
  startingBalance?: number;
  riskPct?: number;
  maxOpenRiskPct?: number;
  useTp2?: boolean;
  tp1ClosePct?: number;
  stopAfterTp1?: 'keep' | 'entry' | 'trail';
  feePct?: number;
  slipPct?: number;
  exchange?: 'binance' | 'coinbase' | 'bybit' | 'nasdaq' | 'bist';
  rangeStart: string;
  rangeEnd: string;
};

export type ScannerRuleListResponse = {
  clientId?: string;
  count?: number;
  rules?: ScannerRule[];
};

export type ScannerSetup = {
  key: string;
  exchange: string;
  symbol: string;
  interval: string;
  factors: ScannerRuleType[];
  score: number;
  grade: 'A' | 'B' | 'C';
  sameBar: boolean;
  latestAt: string;
  summaries: string[];
};

export type ScannerResultListResponse = {
  clientId?: string;
  count?: number;
  total?: number;
  limit?: number;
  offset?: number;
  results?: ScannerResult[];
  setups?: ScannerSetup[];
  hits24h?: number;
};

export type ScannerBacktestListResponse = {
  clientId?: string;
  count?: number;
  total?: number;
  backtests?: ScannerBacktest[];
};

export type ScannerBacktestHorizonStats = {
  days: number;
  sampleCount: number;
  average?: number;
  median?: number;
  winRate?: number;
};

export type ScannerBacktestTradeStats = {
  sampleCount: number;
  tpHits: number;
  slHits: number;
  openCount: number;
  longCount: number;
  shortCount: number;
  winRate?: number;
  average?: number;
  median?: number;
};

export type ScannerBacktestMarketStatus = 'ok' | 'no_data' | 'error';

export type ScannerBacktestMarketPerformance = {
  exchange: string;
  symbol: string;
  status?: ScannerBacktestMarketStatus;
  error?: string;
  signalCount: number;
  return1d: ScannerBacktestHorizonStats;
  return5d: ScannerBacktestHorizonStats;
  return20d: ScannerBacktestHorizonStats;
  trades?: ScannerBacktestTradeStats;
};

export type ScannerBacktestBook = {
  startingBalance: number;
  rangeEndBalance?: number;
  afterRangeBalance?: number;
  endingBalance: number;
  peakEquity?: number;
  troughEquity?: number;
  riskPct?: number;
  maxOpenRiskPct?: number;
  stakePct?: number;
  tradesTaken?: number;
  tradesSkipped?: number;
  skippedNoRisk?: number;
  skippedSameCoin?: number;
  closedInRange?: number;
  closedAfterRange?: number;
  openAtRangeEnd?: number;
  openAtEnd?: number;
  bookLegs?: number;
  bookSlipPct?: number;
};

export type ScannerBacktestPerformance = {
  signalCount: number;
  return1d: ScannerBacktestHorizonStats;
  return5d: ScannerBacktestHorizonStats;
  return20d: ScannerBacktestHorizonStats;
  trades?: ScannerBacktestTradeStats;
  markets?: ScannerBacktestMarketPerformance[];
  book?: ScannerBacktestBook;
};

export type ScannerBacktestSignalListResponse = {
  backtestId?: string;
  count?: number;
  total?: number;
  signalCount?: number;
  signals?: ScannerBacktestSignal[];
  performance?: ScannerBacktestPerformance;
};
