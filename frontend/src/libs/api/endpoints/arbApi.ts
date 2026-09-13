import { baseApi } from '../baseApi';
import type { components, operations } from '../generated/schema';
import { compactParams } from './marketApi.helpers';

export type PriceDiffQuote = components['schemas']['PriceDiffQuote'];
export type PriceDiffQuoteScan = components['schemas']['PriceDiffQuoteScan'];
export type PriceDiffWatch = components['schemas']['PriceDiffWatch'];
export type PriceDiffOpportunity = components['schemas']['PriceDiffOpportunity'];
export type PriceDiffScanQuery = NonNullable<
  operations['scanPriceDiffQuotes']['parameters']['query']
>;
export type CreatePriceDiffWatchArg = operations['createPriceDiffWatch']['requestBody']['content']['application/json'];

export type PriceDiffWatchList = {
  clientId?: string;
  watches?: PriceDiffWatch[];
  count?: number;
};

export type PriceDiffOpportunityList = {
  clientId?: string;
  opportunities?: PriceDiffOpportunity[];
  count?: number;
};

export type MarketFundingArbScan = components['schemas']['MarketFundingArbScan'];
export type MarketFundingArbHit = components['schemas']['MarketFundingArbHit'];
export type FundingArbWatch = components['schemas']['FundingArbWatch'];
export type FundingArbScanQuery = NonNullable<
  operations['scanMarketFundingArb']['parameters']['query']
>;
export type CreateFundingArbWatchArg = {
  symbol?: string;
  quote?: string;
  notional?: number;
  minProfit?: number;
  holdHours?: number;
  feeBinancePct?: number;
  feeBybitPct?: number;
};

export type FundingArbWatchList = {
  clientId?: string;
  watches?: FundingArbWatch[];
  count?: number;
};

export type FundingArbReviewPayment = {
  time?: string;
  exchange?: string;
  expectedRate?: string;
  expectedRatePct?: string;
  actualRate?: string;
  actualRatePct?: string;
  expectedAmount?: string;
  actualAmount?: string;
  status?: string;
};

export type FundingArbReview = {
  asOf?: string;
  pendingCount?: number;
  settledCount?: number;
  missingCount?: number;
  complete?: boolean;
  expectedFunding?: number;
  actualFunding?: number;
  expectedNet?: number;
  actualNet?: number;
  deltaNet?: number;
  stillWorthIt?: boolean;
  summary?: string;
  payments?: FundingArbReviewPayment[];
};

export type FundingArbSignal = {
  id?: string;
  watchId?: string;
  symbol?: string;
  longExchange?: string;
  shortExchange?: string;
  netAfterFees?: number;
  status?: string;
  openedAt?: string;
  expected?: { expectedNet?: number; payments?: FundingArbReviewPayment[] };
  review?: FundingArbReview;
};

export type FundingArbSignalList = {
  signals?: FundingArbSignal[];
};

export type MarketSnapshotWindow = {
  window?: string;
  price?: { changePct?: string; direction?: string; complete?: boolean };
  volume?: { changePct?: string; direction?: string; complete?: boolean };
  marketCap?: { changePct?: string; direction?: string; complete?: boolean };
  openInterest?: { changePct?: string; direction?: string; complete?: boolean };
  funding?: { current?: string; changePct?: string; direction?: string; complete?: boolean };
  longPct?: { current?: string; changePct?: string; direction?: string; complete?: boolean };
  taker?: { buyShare?: string; dominant?: string; complete?: boolean };
};

export type MarketSnapshotVenue = {
  exchange?: string;
  openInterestValue?: string;
  fundingRate?: string;
  longPct?: string;
  windows?: MarketSnapshotWindow[];
  summary?: string;
  error?: string;
};

export type MarketSnapshot = {
  symbol?: string;
  exchange?: string;
  asOf?: string;
  summary?: string;
  note?: string;
  spot?: {
    price?: string;
    volume24h?: string;
    marketCap?: string;
    circulating?: string;
    windows?: MarketSnapshotWindow[];
  };
  venues?: MarketSnapshotVenue[];
  combined?: MarketSnapshotVenue;
};

export const arbApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    scanPriceDiffQuotes: build.query<PriceDiffQuoteScan, PriceDiffScanQuery>({
      query: (arg) => ({
        url: '/api/v1/price-diff/quote/scan',
        params: compactParams({ ...arg }),
      }),
    }),
    listPriceDiffWatches: build.query<PriceDiffWatchList, void>({
      query: () => '/api/v1/price-diff/watches',
      providesTags: ['PriceDiffWatch'],
    }),
    createPriceDiffWatch: build.mutation<PriceDiffWatch, CreatePriceDiffWatchArg>({
      query: (body) => ({ url: '/api/v1/price-diff/watches', method: 'POST', body }),
      invalidatesTags: ['PriceDiffWatch'],
    }),
    pausePriceDiffWatch: build.mutation<PriceDiffWatch, string>({
      query: (id) => ({ url: `/api/v1/price-diff/watches/${id}/pause`, method: 'POST' }),
      invalidatesTags: ['PriceDiffWatch', 'PriceDiffOpportunity'],
    }),
    resumePriceDiffWatch: build.mutation<PriceDiffWatch, string>({
      query: (id) => ({ url: `/api/v1/price-diff/watches/${id}/resume`, method: 'POST' }),
      invalidatesTags: ['PriceDiffWatch'],
    }),
    deletePriceDiffWatch: build.mutation<void, string>({
      query: (id) => ({ url: `/api/v1/price-diff/watches/${id}`, method: 'DELETE' }),
      invalidatesTags: ['PriceDiffWatch', 'PriceDiffOpportunity'],
    }),
    listPriceDiffOpportunities: build.query<PriceDiffOpportunityList, { status?: string } | void>({
      query: (arg) => ({
        url: '/api/v1/price-diff/opportunities',
        params: compactParams({ status: arg && 'status' in arg ? arg.status : 'open' }),
      }),
      providesTags: ['PriceDiffOpportunity'],
    }),
    scanMarketFundingArb: build.query<MarketFundingArbScan, FundingArbScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/funding-arb/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    listFundingArbWatches: build.query<FundingArbWatchList, void>({
      query: () => '/api/v1/funding-arb/watches',
      providesTags: ['FundingArbWatch'],
    }),
    createFundingArbWatch: build.mutation<FundingArbWatch, CreateFundingArbWatchArg>({
      query: (body) => ({ url: '/api/v1/funding-arb/watches', method: 'POST', body }),
      invalidatesTags: ['FundingArbWatch'],
    }),
    deleteFundingArbWatch: build.mutation<void, string>({
      query: (id) => ({ url: `/api/v1/funding-arb/watches/${id}`, method: 'DELETE' }),
      invalidatesTags: ['FundingArbWatch'],
    }),
    listFundingArbSignals: build.query<FundingArbSignalList, { status?: string; limit?: number } | void>({
      query: (arg) => ({
        url: '/api/v1/funding-arb/signals',
        params: compactParams({
          status: arg && 'status' in arg ? arg.status : 'all',
          limit: arg && 'limit' in arg ? arg.limit : 20,
        }),
      }),
      providesTags: ['FundingArbWatch'],
    }),
    getFundingArbSignal: build.query<FundingArbSignal, string>({
      query: (id) => `/api/v1/funding-arb/signals/${id}`,
      providesTags: ['FundingArbWatch'],
    }),
    getMarketSnapshot: build.query<MarketSnapshot, { symbol: string; exchange?: string }>({
      query: (arg) => ({
        url: '/api/v1/market/snapshot',
        params: compactParams({ ...arg }),
      }),
    }),
  }),
});

export const {
  useScanPriceDiffQuotesQuery,
  useLazyScanPriceDiffQuotesQuery,
  useListPriceDiffWatchesQuery,
  useCreatePriceDiffWatchMutation,
  usePausePriceDiffWatchMutation,
  useResumePriceDiffWatchMutation,
  useDeletePriceDiffWatchMutation,
  useListPriceDiffOpportunitiesQuery,
  useScanMarketFundingArbQuery,
  useLazyScanMarketFundingArbQuery,
  useListFundingArbWatchesQuery,
  useCreateFundingArbWatchMutation,
  useDeleteFundingArbWatchMutation,
  useListFundingArbSignalsQuery,
  useGetFundingArbSignalQuery,
  useGetMarketSnapshotQuery,
} = arbApi;
