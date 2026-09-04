export type ForensicsExchange = 'all' | 'binance' | 'bybit';
export type ForensicsWindow = '15m' | '30m' | '1h' | '2h' | '4h';
export type ForensicsDuring = '5m' | '15m' | '30m' | '1h';
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

export type ForensicsTape = {
  symbol?: string;
  exchange?: string;
  at?: string;
  window?: string;
  during?: string;
  venues?: ForensicsVenue[];
  combined?: ForensicsVenue | null;
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
  summary?: string;
  note?: string;
};
