import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type PriceAlert = components['schemas']['PriceAlert'];

type AlertListResponse = {
  clientId?: string;
  count?: number;
  alerts?: PriceAlert[];
};

export type CreatePriceAlertArg = {
  exchange?: string;
  symbol?: string;
  kind?: 'price' | 'imbalance' | 'wall' | 'liquidation_feed' | 'liquidation_cascade';
  condition?: string;
  targetPrice?: number;
  mode?: 'one_time' | 'repeating';
};

export const alertsApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listPriceAlerts: build.query<AlertListResponse, void>({
      query: () => '/api/v1/alerts',
      providesTags: ['Alert'],
    }),
    createPriceAlert: build.mutation<PriceAlert, CreatePriceAlertArg>({
      query: (body) => ({
        url: '/api/v1/alerts',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['Alert'],
    }),
    deletePriceAlert: build.mutation<{ id?: string }, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/alerts/${encodeURIComponent(id)}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['Alert'],
    }),
  }),
});

export const {
  useListPriceAlertsQuery,
  useCreatePriceAlertMutation,
  useDeletePriceAlertMutation,
} = alertsApi;
