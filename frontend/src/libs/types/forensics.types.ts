export type ForensicsExchange = 'all' | 'binance' | 'bybit' | 'okx';
export type ForensicsWindow = '15m' | '30m' | '1h' | '2h' | '4h';
export type ForensicsDuring = '5m' | '15m' | '30m' | '1h';
export type ForensicsView = 'event' | 'now' | 'review';
export type ForensicsLookback = '4h' | '12h' | '24h' | '3d' | '7d';
export type ForensicsPhaseId = 'before' | 'during' | 'after';

export type ForensicsPrice = {
  open?: string;
  high?: string;
  low?: string;
  close?: string;
  change?: string;
  changePct?: string;
  range?: string;
  rangePct?: string;
  direction?: string;
};

export type ForensicsFlow = {
  volume?: string;
  buyVolume?: string;
  sellVolume?: string;
  delta?: string;
  buyShare?: string;
  dominant?: string;
  vwap?: string;
  vsVWAP?: string;
  volumeRatio?: string;
  volumeGrade?: string;
  typicalKnown?: boolean;
};

export type ForensicsProfile = {
  poc?: string;
  lastVsArea?: string;
};

export type ForensicsBook = {
  midDelta?: string;
  midDeltaPct?: string;
  summary?: string;
  complete?: boolean;
};

export type ForensicsFutures = {
  oiChange?: string;
  oiChangePct?: string;
  oiDirection?: string;
  fundingFrom?: string;
  fundingTo?: string;
  longLiq?: string;
  shortLiq?: string;
  complete?: boolean;
};

export type ForensicsOptionsMove = {
  kind?: string;
  topic?: string;
  window?: string;
  title?: string;
  detail?: string;
};

export type ForensicsOptions = {
  asset?: string;
  oiChangePct?: string;
  oiDirection?: string;
  putCallFrom?: string;
  putCallTo?: string;
  leanFrom?: string;
  leanTo?: string;
  dvolDelta?: string;
  unusual?: boolean;
  started?: 'before' | 'during' | string;
  startedAt?: string;
  lead?: string;
  leadLabel?: string;
  leadWindow?: string;
  together?: number;
  weight?: string;
  after?: string;
  afterSummary?: string;
  summary?: string;
  moves?: ForensicsOptionsMove[];
  complete?: boolean;
};

export type ForensicsCurve = {
  exchange?: string;
  shape?: string;
  slope?: string;
  spread?: string;
  spreadPct?: string;
  nearLabel?: string;
  farLabel?: string;
  nearMark?: string;
  farMark?: string;
  at?: string;
  complete?: boolean;
  summary?: string;
};

export type ForensicsEvent = {
  kind?: string;
  phase?: string;
  side?: string;
  title?: string;
  summary?: string;
  at?: string;
  score?: number;
};

export type ForensicsPhase = {
  phase?: string;
  from?: string;
  to?: string;
  barCount?: number;
  price?: ForensicsPrice;
  flow?: ForensicsFlow;
  profile?: ForensicsProfile | null;
  book?: ForensicsBook | null;
  futures?: ForensicsFutures | null;
  options?: ForensicsOptions | null;
  curve?: ForensicsCurve | null;
  events?: ForensicsEvent[];
  summary?: string;
  complete?: boolean;
};

export type ForensicsChange = {
  metric?: string;
  before?: string;
  during?: string;
  after?: string;
  path?: string;
  direction?: string;
  summary?: string;
};

export type ForensicsVenue = {
  exchange?: string;
  symbol?: string;
  interval?: string;
  phases?: ForensicsPhase[];
  changes?: ForensicsChange[];
  events?: ForensicsEvent[];
  summary?: string;
  error?: string;
};

export type ForensicsVenueBeforeBit = {
  metric?: string;
  venue?: string;
  direction?: string;
  phase?: string;
  at?: string;
  until?: string;
  summary?: string;
};

export type ForensicsVenueTiming = {
  agreement?: string;
  follow?: string;
  leader?: string;
  leaderMetric?: string;
  binanceStart?: string;
  bybitStart?: string;
  binanceOiStart?: string;
  bybitOiStart?: string;
  binanceTakerStart?: string;
  bybitTakerStart?: string;
  lead?: string;
  leadLabel?: string;
  beforeFollow?: ForensicsVenueBeforeBit[];
  summary?: string;
};

export type ForensicsTape = {
  symbol?: string;
  exchange?: string;
  at?: string;
  window?: string;
  during?: string;
  venues?: ForensicsVenue[];
  combined?: ForensicsVenue | null;
  venueTiming?: ForensicsVenueTiming | null;
  summary?: string;
};

export type ForensicsSimilarHit = {
  at?: string;
  until?: string;
  direction?: string;
  returnPct?: string;
  grade?: string;
  similarity?: string;
  coverage?: string;
  afterReturnPct?: string;
  afterDirection?: string;
  summary?: string;
};

export type ForensicsHorizon = {
  horizon?: string;
  sample?: number;
  events?: number;
  up?: number;
  down?: number;
  averagePct?: string;
  medianPct?: string;
};

export type ForensicsSimilar = {
  at?: string;
  summary?: string;
  matches?: ForensicsSimilarHit[];
  afterHorizons?: ForensicsHorizon[];
  events?: number;
};

export type ForensicsWhy = {
  headline?: string;
  before?: string;
  during?: string;
  after?: string;
  similar?: string;
  caution?: string;
};

export type ForensicsTimelineEvent = {
  at?: string;
  offset?: string;
  offsetLabel?: string;
  phase?: string;
  kind?: string;
  venue?: string;
  direction?: string;
  title?: string;
  detail?: string;
  resolution?: string;
  orderUnknown?: boolean;
};

export type ForensicsTimeline = {
  events?: ForensicsTimelineEvent[];
  summary?: string;
  knownUntil?: string;
  orderKnown?: boolean;
};

export type ForensicsReport = {
  symbol?: string;
  exchange?: string;
  at?: string;
  window?: string;
  during?: string;
  asOf?: string;
  tape?: ForensicsTape;
  similar?: ForensicsSimilar | null;
  similarError?: string;
  why?: ForensicsWhy | null;
  timeline?: ForensicsTimeline | null;
  summary?: string;
  note?: string;
};

export type ForensicsScanField = {
  name?: string;
  used?: boolean;
  score?: string;
  weight?: string;
};

export type ForensicsScanHorizon = {
  horizon?: string;
  sample?: number;
  events?: number;
  up?: number;
  down?: number;
  averagePct?: string;
  medianPct?: string;
};

export type ForensicsScanHit = {
  symbol?: string;
  exchange?: string;
  similarity?: string;
  coverage?: string;
  events?: number;
  matches?: number;
  upAfter?: number;
  downAfter?: number;
  medianAfterPct?: string;
  afterHorizons?: ForensicsScanHorizon[];
  closestAt?: string;
  closestDirection?: string;
  closestReturnPct?: string;
  lean?: string;
  compared?: ForensicsScanField[];
  whySimilar?: string;
  afterStory?: string;
  summary?: string;
};

export type ForensicsRadarAfter = {
  horizon?: string;
  dueAt?: string;
  filledAt?: string;
  returnPct?: string;
  direction?: string;
  price?: string;
  useful?: boolean | null;
};

export type ForensicsRadarCase = {
  id?: string;
  exchange?: string;
  symbol?: string;
  foundAt?: string;
  foundPrice?: string;
  similarity?: string;
  lean?: string;
  whySimilar?: string;
  afterStory?: string;
  after?: ForensicsRadarAfter[];
};

export type ForensicsRadarHorizon = {
  horizon?: string;
  sample?: number;
  leanRight?: number;
  leanWrong?: number;
  up?: number;
  down?: number;
  flat?: number;
  medianPct?: string;
  averagePct?: string;
  usefulPct?: string;
};

export type ForensicsRadarReview = {
  exchange?: string;
  symbol?: string;
  since?: string;
  summary?: string;
  note?: string;
  cases?: ForensicsRadarCase[];
  horizons?: ForensicsRadarHorizon[];
};

export type ForensicsScan = {
  exchange?: string;
  quote?: string;
  lookback?: string;
  window?: string;
  scanned?: number;
  compared?: number;
  hits?: ForensicsScanHit[];
  summary?: string;
  note?: string;
};
