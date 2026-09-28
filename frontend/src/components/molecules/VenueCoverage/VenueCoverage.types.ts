export type VenueRead = {
  exchange?: string;
  status?: string;
  asOf?: string;
  ageSec?: number;
};

export type MarketVenueCoverage = {
  complete?: boolean;
  venues?: VenueRead[];
  missing?: string[];
  stale?: string[];
};

export type VenueCoverageProps = {
  coverage?: MarketVenueCoverage | null;
};
