export type MaxPainPocket = {
  exchange?: string;
  side?: string;
  direction?: string;
  price?: string;
  movePct?: string;
  notional?: string;
  estNotional?: string;
  observedNotional?: string;
  leverage?: string;
  source?: string;
};

export type MaxPainVenue = {
  exchange?: string;
  symbol?: string;
  price?: string;
  openInterestValue?: string;
  above?: MaxPainPocket;
  below?: MaxPainPocket;
  aboveLevels?: MaxPainPocket[];
  belowLevels?: MaxPainPocket[];
  error?: string;
};

export type MaxPainReport = {
  symbol?: string;
  exchange?: string;
  asOf?: string;
  above?: MaxPainPocket;
  below?: MaxPainPocket;
  venues?: MaxPainVenue[];
  summary?: string;
  note?: string;
};

export type LiquidationMaxPainProps = {
  data?: MaxPainReport | null;
  isLoading?: boolean;
  errorMessage?: string | null;
};
