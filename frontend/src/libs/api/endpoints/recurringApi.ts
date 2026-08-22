import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type RecurringBuyPlan = components['schemas']['RecurringBuyPlan'];

type RecurringListResponse = {
  clientId?: string;
  count?: number;
  plans?: RecurringBuyPlan[];
};

export type CreateRecurringBuyArg = {
  exchange?: string;
  symbol: string;
  amount: number;
  frequency: 'daily' | 'weekly' | 'monthly' | 'interval';
  name?: string;
  weekday?: RecurringBuyPlan['weekday'];
  dayOfMonth?: number;
  intervalHours?: number;
};

export const recurringApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listRecurringBuys: build.query<RecurringListResponse, void>({
      query: () => '/api/v1/portfolio/recurring-buys',
      providesTags: ['RecurringBuy'],
    }),
    createRecurringBuy: build.mutation<RecurringBuyPlan, CreateRecurringBuyArg>({
      query: (body) => ({ url: '/api/v1/portfolio/recurring-buys', method: 'POST', body }),
      invalidatesTags: ['RecurringBuy'],
    }),
    pauseRecurringBuy: build.mutation<RecurringBuyPlan, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/portfolio/recurring-buys/${encodeURIComponent(id)}/pause`,
        method: 'POST',
      }),
      invalidatesTags: ['RecurringBuy'],
    }),
    resumeRecurringBuy: build.mutation<RecurringBuyPlan, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/portfolio/recurring-buys/${encodeURIComponent(id)}/resume`,
        method: 'POST',
      }),
      invalidatesTags: ['RecurringBuy'],
    }),
    deleteRecurringBuy: build.mutation<{ deleted?: boolean }, { id: string }>({
      query: ({ id }) => ({
        url: `/api/v1/portfolio/recurring-buys/${encodeURIComponent(id)}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['RecurringBuy'],
    }),
  }),
});

export const {
  useListRecurringBuysQuery,
  useCreateRecurringBuyMutation,
  usePauseRecurringBuyMutation,
  useResumeRecurringBuyMutation,
  useDeleteRecurringBuyMutation,
} = recurringApi;
