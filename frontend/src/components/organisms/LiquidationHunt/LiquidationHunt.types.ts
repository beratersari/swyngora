export type HuntLean = 'up' | 'down' | 'even' | string;
export type HuntEase = 'easier' | 'likely' | 'mixed' | 'hard' | string;
export type HuntHouseEdge = 'profit' | 'loss' | 'unreachable' | string;

export type HuntFactor = {
  id?: string;
  label?: string;
  score?: number;
  weight?: number;
  detail?: string;
};

export type HuntCoverageLevel = 'complete' | 'usable' | 'thin' | 'insufficient' | string;
export type HuntInputState = 'ok' | 'weak' | 'missing' | 'error' | string;

export type HuntInputStatus = {
  id?: string;
  label?: string;
  status?: HuntInputState;
  weight?: number;
  detail?: string;
};

export type HuntCoverage = {
  score?: number;
  level?: HuntCoverageLevel;
  usable?: boolean;
  inputs?: HuntInputStatus[];
  missing?: string[];
  weak?: string[];
  summary?: string;
};

export type HuntDirectionScore = {
  direction?: string;
  score?: number;
  level?: HuntEase;
  coverage?: number;
  factors?: HuntFactor[];
  reasons?: string[];
};

export type HuntBias = {
  lean?: HuntLean;
  margin?: number;
  upScore?: number;
  downScore?: number;
  summary?: string;
  coverage?: HuntCoverage;
  included?: string[];
  excluded?: string[];
};

export type HuntBand = {
  side?: string;
  direction?: string;
  leverage?: string;
  price?: string;
  movePct?: string;
  estNotional?: string;
  observedNotional?: string;
  source?: string;
};

export type HuntWalk = {
  side?: string;
  targetPrice?: string;
  quantity?: string;
  notional?: string;
  averagePrice?: string;
  endPrice?: string;
  reachable?: boolean;
  exhausted?: boolean;
  maxReachablePrice?: string;
  visibleNotional?: string;
};

export type HuntScenario = {
  direction?: string;
  thesis?: string;
  target?: HuntBand;
  spot?: HuntWalk;
  estLiquidated?: string;
  cascadeExitNotional?: string;
  bookOnlyPnl?: string;
  cascadeInventoryPnl?: string;
  liquidationTake?: string;
  fees?: string;
  netBookOnly?: string;
  netWithCascade?: string;
  houseEdge?: HuntHouseEdge;
  efficiency?: string;
};

export type HuntVenue = {
  exchange?: string;
  symbol?: string;
  price?: string;
  openInterestValue?: string;
  estLongNotional?: string;
  estShortNotional?: string;
  longPct?: string;
  shortPct?: string;
  estLongPct?: string;
  estShortPct?: string;
  fundingRate?: string;
  fundingPayer?: string;
  visibleBidNotional?: string;
  visibleAskNotional?: string;
  upPressure?: HuntBand[];
  downPressure?: HuntBand[];
  upHunt?: HuntScenario;
  downHunt?: HuntScenario;
  upScore?: HuntDirectionScore;
  downScore?: HuntDirectionScore;
  bias?: HuntBias;
  coverage?: HuntCoverage;
  error?: string;
};

export type HuntReport = {
  symbol?: string;
  exchange?: string;
  asOf?: string;
  venues?: HuntVenue[];
  bias?: HuntBias;
  coverage?: HuntCoverage;
  note?: string;
};

export type LiquidationHuntProps = {
  data?: HuntReport | null;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
};
