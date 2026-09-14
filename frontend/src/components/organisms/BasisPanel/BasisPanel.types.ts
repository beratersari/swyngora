export type BasisKind = 'premium' | 'discount' | 'flat' | string;
export type BasisTrend = 'expanding' | 'shrinking' | 'stable' | 'flipped' | string;
export type BasisShape = 'contango' | 'backwardation' | 'flat' | 'humped' | 'unknown' | string;
export type BasisSlope = 'up' | 'down' | 'flat' | string;
export type BasisRicher = 'near' | 'far' | 'none' | string;

export type BasisLevel = {
  futures?: string;
  spot?: string;
  delta?: string;
  deltaPct?: string;
  kind?: BasisKind;
  source?: string;
};

export type BasisWindow = {
  window?: string;
  pastPct?: string;
  changePct?: string;
  trend?: BasisTrend;
  complete?: boolean;
};

export type BasisContract = {
  symbol?: string;
  kind?: string;
  label?: string;
  expiry?: string;
  daysToExpiry?: number;
  last?: string;
  mark?: string;
  index?: string;
  vsIndex?: string;
  vsIndexPct?: string;
};

export type BasisCurve = {
  points?: BasisContract[];
  near?: BasisContract;
  far?: BasisContract;
  spread?: string;
  spreadPct?: string;
  shape?: BasisShape;
  slope?: BasisSlope;
  richer?: BasisRicher;
  summary?: string;
  complete?: boolean;
};

export type BasisVenue = {
  exchange?: string;
  symbol?: string;
  last?: BasisLevel;
  mark?: BasisLevel;
  windows?: BasisWindow[];
  trend?: BasisTrend;
  curve?: BasisCurve;
  summary?: string;
  error?: string;
};

export type BasisAgreement = {
  alignment?: string;
  title?: string;
  summary?: string;
};

export type BasisReport = {
  symbol?: string;
  exchange?: string;
  asOf?: string;
  venues?: BasisVenue[];
  agreement?: BasisAgreement;
  note?: string;
};

export type BasisPanelProps = {
  report?: BasisReport | null;
  isLoading?: boolean;
  errorMessage?: string | null;
};
