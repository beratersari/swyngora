import { baseApi } from '../baseApi';
import { compactParams } from './marketApi';

export type PumpDirection = 'up' | 'down' | 'both';
export type PumpMode = 'close_return' | 'candle_body' | 'high_from_low';

export type PumpEvent = {
  index?: number;
  openTime?: string;
  closeTime?: string;
  startPrice?: number;
  endPrice?: number;
  returnPct?: number;
  high?: number;
  low?: number;
  volume?: number;
  volumeRatio?: number;
  mode?: string;
  windowBars?: number;
};

export type PumpEventsResponse = {
  exchange?: string;
  symbol?: string;
  interval?: string;
  lookbackHours?: number;
  barsAnalyzed?: number;
  minReturnPct?: number;
  windowBars?: number;
  mode?: string;
  direction?: string;
  events?: PumpEvent[];
  eventCount?: number;
  note?: string;
};

export type PumpScanHit = {
  symbol?: string;
  exchange?: string;
  interval?: string;
  bestReturnPct?: number;
  events?: PumpEvent[];
};

export type PumpScanResponse = {
  exchange?: string;
  interval?: string;
  lookbackHours?: number;
  minReturnPct?: number;
  hits?: PumpScanHit[];
  hitCount?: number;
  note?: string;
};

export type ScanPumpEventsQuery = {
  exchange?: string;
  quote?: string;
  interval?: string;
  lookbackHours?: number;
  minReturnPct?: number;
  windowBars?: number;
  mode?: string;
  direction?: string;
  minVolumeRatio?: number;
  symbolLimit?: number;
  maxTotalEvents?: number;
};

export type GetPumpEventsQuery = {
  symbol: string;
  exchange?: string;
  interval?: string;
  lookbackHours?: number;
  limit?: number;
  minReturnPct?: number;
  windowBars?: number;
  mode?: string;
  direction?: string;
  minVolumeRatio?: number;
  maxEvents?: number;
};

export const pumpApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    scanPumpEvents: build.query<PumpScanResponse, ScanPumpEventsQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/pumps/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Pump' as const,
          id: `scan:${arg?.exchange ?? 'binance'}:${arg?.quote ?? ''}:${arg?.interval ?? ''}:${arg?.lookbackHours ?? ''}:${arg?.minReturnPct ?? ''}:${arg?.direction ?? ''}`,
        },
      ],
    }),

    getPumpEvents: build.query<PumpEventsResponse, GetPumpEventsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/pumps',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Pump' as const,
          id: `sym:${arg.exchange ?? 'binance'}:${arg.symbol}:${arg.interval ?? '1h'}`,
        },
      ],
    }),
  }),
});

export const { useScanPumpEventsQuery, useGetPumpEventsQuery } = pumpApi;
