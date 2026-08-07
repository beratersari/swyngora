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

export type CandlesQuery = NonNullable<operations['getCandles']['parameters']['query']>;
export type Ticker24hQuery = NonNullable<operations['getTicker24h']['parameters']['query']>;
export type SupplyQuery = NonNullable<operations['getSupply']['parameters']['query']>;
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
