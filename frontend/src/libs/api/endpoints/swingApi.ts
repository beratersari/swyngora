import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type SwingDecision = components['schemas']['SwingDecision'];
export type SwingLevels = components['schemas']['SwingLevels'];
export type SwingPattern = components['schemas']['SwingPattern'];
export type SwingSetupList = components['schemas']['SwingSetupList'];
export type CustomSignalProfile = components['schemas']['CustomSignalProfile'];
export type CustomSignalProfileList = components['schemas']['CustomSignalProfileList'];
export type CustomSignalItem = components['schemas']['CustomSignalItem'];
export type CustomSignalScan = components['schemas']['CustomSignalScan'];
export type CustomSignalFactorList = components['schemas']['CustomSignalFactorList'];
export type CustomSignalProfileWrite = components['schemas']['CustomSignalProfileWrite'];

export type ScanCustomSignalsArg = {
  profileId?: string;
  exchange?: string;
  limit?: number;
  weightEmaCrossover?: number;
  weightEmaTrend?: number;
  weightSupertrend?: number;
  weightRsiRecovery?: number;
  weightMacd?: number;
  weightAdx?: number;
  weightAccumulation?: number;
  weightVolumeBreakout?: number;
  weightHuntUp?: number;
};

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
    listCustomSignalFactors: build.query<CustomSignalFactorList, void>({
      query: () => '/api/v1/swing/custom/factors',
    }),
    listCustomSignalProfiles: build.query<CustomSignalProfileList, void>({
      query: () => '/api/v1/swing/custom/profiles',
      providesTags: ['CustomSignalProfile'],
    }),
    createCustomSignalProfile: build.mutation<CustomSignalProfile, CustomSignalProfileWrite>({
      query: (body) => ({
        url: '/api/v1/swing/custom/profiles',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['CustomSignalProfile'],
    }),
    updateCustomSignalProfile: build.mutation<
      CustomSignalProfile,
      { id: string } & CustomSignalProfileWrite
    >({
      query: ({ id, ...body }) => ({
        url: `/api/v1/swing/custom/profiles/${encodeURIComponent(id)}`,
        method: 'PATCH',
        body,
      }),
      invalidatesTags: ['CustomSignalProfile', 'CustomSignalScan'],
    }),
    deleteCustomSignalProfile: build.mutation<void, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/swing/custom/profiles/${encodeURIComponent(id)}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['CustomSignalProfile', 'CustomSignalScan'],
    }),
    duplicateCustomSignalProfile: build.mutation<CustomSignalProfile, { id: string; name?: string }>({
      query: ({ id, name }) => ({
        url: `/api/v1/swing/custom/profiles/${encodeURIComponent(id)}/duplicate`,
        method: 'POST',
        body: name ? { name } : {},
      }),
      invalidatesTags: ['CustomSignalProfile'],
    }),
    setDefaultCustomSignalProfile: build.mutation<CustomSignalProfile, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/swing/custom/profiles/${encodeURIComponent(id)}/default`,
        method: 'POST',
      }),
      invalidatesTags: ['CustomSignalProfile'],
    }),
    scanCustomSignals: build.query<CustomSignalScan, ScanCustomSignalsArg | void>({
      query: (arg) => ({
        url: '/api/v1/swing/custom/scan',
        params: {
          profileId: arg?.profileId,
          exchange: arg?.exchange,
          limit: arg?.limit ?? 25,
          weightEmaCrossover: arg?.weightEmaCrossover,
          weightEmaTrend: arg?.weightEmaTrend,
          weightSupertrend: arg?.weightSupertrend,
          weightRsiRecovery: arg?.weightRsiRecovery,
          weightMacd: arg?.weightMacd,
          weightAdx: arg?.weightAdx,
          weightAccumulation: arg?.weightAccumulation,
          weightVolumeBreakout: arg?.weightVolumeBreakout,
          weightHuntUp: arg?.weightHuntUp,
        },
      }),
      providesTags: ['CustomSignalScan'],
    }),
  }),
});

export const {
  useAnalyzeSwingQuery,
  useListSwingSetupsQuery,
  useListCustomSignalFactorsQuery,
  useListCustomSignalProfilesQuery,
  useCreateCustomSignalProfileMutation,
  useUpdateCustomSignalProfileMutation,
  useDeleteCustomSignalProfileMutation,
  useDuplicateCustomSignalProfileMutation,
  useSetDefaultCustomSignalProfileMutation,
  useScanCustomSignalsQuery,
} = swingApi;
