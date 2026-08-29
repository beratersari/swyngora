import type { components, operations } from '../generated/schema';

export type SpotMarket = components['schemas']['SpotMarket'];
export type SpotListResponse = components['schemas']['SpotListResponse'];
export type SpotListQuery = NonNullable<operations['listSpotMarkets']['parameters']['query']>;
export type SpotSortField = NonNullable<SpotListQuery['sort']>;
export type SpotSortOrder = NonNullable<SpotListQuery['order']>;
export type MarketExchange = NonNullable<SpotListQuery['exchange']>;

export type CandlesResponse = components['schemas']['CandlesResponse'];
export type Candle = components['schemas']['Candle'];
export type Ticker24h = components['schemas']['Ticker24h'];
export type Supply = components['schemas']['Supply'];
export type AssetHolders = components['schemas']['AssetHolders'];
export type HolderWallet = components['schemas']['HolderWallet'];
export type SpotOrderBook = components['schemas']['SpotOrderBook'];
export type OrderBookLevel = components['schemas']['OrderBookLevel'];
export type OrderBookQuery = NonNullable<operations['getSpotOrderBook']['parameters']['query']>;
export type OrderBookHeatmap = components['schemas']['OrderBookHeatmap'];
export type OrderBookHeatmapQuery = NonNullable<
  operations['getSpotOrderBookHeatmap']['parameters']['query']
>;
export type LiquidationHuntHeatmap = components['schemas']['LiquidationHuntHeatmap'];
export type LiquidationHuntHeatmapGrid = components['schemas']['LiquidationHuntHeatmapGrid'];
export type LiquidationHuntHeatmapQuery = NonNullable<
  operations['getMarketLiquidationHuntHeatmap']['parameters']['query']
>;

export type CandlesQuery = NonNullable<operations['getCandles']['parameters']['query']>;
export type Ticker24hQuery = NonNullable<operations['getTicker24h']['parameters']['query']>;
export type SupplyQuery = NonNullable<operations['getSupply']['parameters']['query']>;
export type HoldersQuery = NonNullable<operations['getHolders']['parameters']['query']>;
export type AssetProfile = components['schemas']['AssetProfile'];
export type AssetProfileQuery = NonNullable<operations['getAssetProfile']['parameters']['query']>;
export type MarketOpenInterest = components['schemas']['MarketOpenInterest'];
export type MarketFundingRate = components['schemas']['MarketFundingRate'];
export type MarketLiquidations = components['schemas']['MarketLiquidations'];
export type OpenInterestQuery = NonNullable<
  operations['getMarketOpenInterest']['parameters']['query']
>;
export type LiquidationsQuery = NonNullable<
  operations['getMarketLiquidations']['parameters']['query']
>;
export type CvdQuery = NonNullable<operations['getMarketCVD']['parameters']['query']>;

export type MarketCvdVenue = {
  exchange?: string;
  lastCvd?: string;
  lastPrice?: string;
  summary?: string;
  error?: string;
  complete?: boolean;
  windows?: { window?: string; cvdChange?: string; label?: string; complete?: boolean }[];
};

export type MarketCvd = {
  symbol?: string;
  exchange?: string;
  summary?: string;
  note?: string;
  venues?: MarketCvdVenue[];
  combined?: MarketCvdVenue;
  spotCombined?: MarketCvdVenue;
  spotFutures?: { summary?: string };
};
export type IntervalsQuery = NonNullable<operations['listIntervals']['parameters']['query']>;
export type IndicatorsQuery = NonNullable<operations['getIndicators']['parameters']['query']>;
export type PumpEventsQuery = NonNullable<operations['getPumpEvents']['parameters']['query']>;
export type ScanPumpEventsQuery = NonNullable<
  operations['scanPumpEvents']['parameters']['query']
>;

export type IntervalsResponse = {
  exchange: string;
  intervals: string[];
};

/** Prefer OpenAPI 200 body when present; fall back to stable hand shape. */
export type IndicatorsResponse = {
  exchange?: string;
  symbol?: string;
  interval?: string;
  rsiPeriod?: number;
  emaPeriods?: number[];
  latest?: {
    rsi?: number | null;
    ema?: Record<string, number>;
  };
  points?: {
    openTime?: string;
    close?: number;
    rsi?: number | null;
    ema?: Record<string, number>;
  }[];
  note?: string;
};

export type ExchangesResponse = {
  exchanges: string[];
  default: string;
};

export type FxRatesResponse = {
  base?: string;
  asOf?: string;
  rates?: Record<string, number>;
  stale?: boolean;
  note?: string;
};

export type ProductTagsResponse = {
  exchange: string;
  tags: string[];
};

/** OpenAPI-generated pump shapes (schema.d.ts). */
export type PumpEventDto = components['schemas']['PumpEvent'];
export type PumpEventsResponse = components['schemas']['PumpEventsResponse'];
export type PumpScanHitDto = components['schemas']['PumpScanHit'];
export type ScanPumpEventsResponse = components['schemas']['PumpScanResponse'];

export type DelistScheduleResponse = components['schemas']['DelistScheduleResponse'];
export type DelistScheduleItem = components['schemas']['DelistScheduleItem'];
export type PostDelistResponse = components['schemas']['PostDelistResponse'];
export type PostDelistQuery = NonNullable<operations['getPostDelist']['parameters']['query']>;
