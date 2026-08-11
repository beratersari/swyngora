import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type SwingDecision = components['schemas']['SwingDecision'];
export type SwingLevels = components['schemas']['SwingLevels'];
export type SwingPattern = components['schemas']['SwingPattern'];
export type SwingSetupList = components['schemas']['SwingSetupList'];

export const swingApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    analyzeSwing: build.query<SwingDecision, { exchange?: string; symbol: string }>({
      query: ({ exchange, symbol }) => ({
        url: '/api/v1/market/swing',
        params: { exchange, symbol },
      }),
    }),
    listSwingSetups: build.query<SwingSetupList, { exchange?: string; limit?: number } | void>({
      query: (arg) => ({
        url: '/api/v1/swing/setups',
        params: {
          exchange: arg?.exchange,
          limit: arg?.limit ?? 25,
        },
      }),
      providesTags: ['SwingSetup'],
    }),
  }),
});

export const { useAnalyzeSwingQuery, useListSwingSetupsQuery } = swingApi;
