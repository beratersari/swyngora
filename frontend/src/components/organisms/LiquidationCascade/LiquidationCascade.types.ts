export type CascadeGrade = 'quiet' | 'elevated' | 'cascade' | 'extreme' | string;
export type CascadeSide = 'long' | 'short' | 'both' | 'none' | string;

export type CascadeWindow = {
  window?: string;
  longNotional?: string;
  shortNotional?: string;
  totalNotional?: string;
  longTypical?: string;
  shortTypical?: string;
  longRatio?: number;
  shortRatio?: number;
  maxRatio?: number;
  side?: CascadeSide;
  grade?: CascadeGrade;
  count?: number;
  sampleBuckets?: number;
  complete?: boolean;
};

export type CascadeVenue = {
  exchange?: string;
  symbol?: string;
  windows?: CascadeWindow[];
  side?: CascadeSide;
  grade?: CascadeGrade;
  score?: number;
  hottest?: string;
  startedAt?: string;
  summary?: string;
};

export type CascadeBoth = {
  agree?: boolean;
  side?: CascadeSide;
  grade?: CascadeGrade;
  score?: number;
  hottest?: string;
  summary?: string;
};

export type CascadeReport = {
  symbol?: string;
  exchange?: string;
  asOf?: string;
  venues?: CascadeVenue[];
  both?: CascadeBoth;
  summary?: string;
  note?: string;
};

export type CascadeHit = {
  symbol?: string;
  side?: CascadeSide;
  grade?: CascadeGrade;
  score?: number;
  hottest?: string;
  both?: boolean;
  summary?: string;
};

export type LiquidationCascadeProps = {
  report?: CascadeReport | null;
  hits?: CascadeHit[];
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
  onOpenCoin?: (symbol: string) => void;
};
