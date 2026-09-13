import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type RecurringBuyPlan = components['schemas']['RecurringBuyPlan'];
export type RecurringBuyRun = components['schemas']['RecurringBuyRun'];

type RecurringRunsResponse = {
  planId?: string;
  count?: number;
  runs?: RecurringBuyRun[];
};

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
    listRecurringBuyRuns: build.query<RecurringRunsResponse, { id: string }>({
      query: ({ id }) => `/api/v1/portfolio/recurring-buys/${encodeURIComponent(id)}/runs`,
      providesTags: (_r, _e, arg) => [{ type: 'RecurringBuy', id: `runs-${arg.id}` }],
    }),
    completeRecurringBuyRun: build.mutation<
      { run?: RecurringBuyRun; tradeId?: string },
      { planId: string; runId: string }
    >({
      query: ({ planId, runId }) => ({
        url: `/api/v1/portfolio/recurring-buys/${encodeURIComponent(planId)}/runs/${encodeURIComponent(runId)}/complete`,
        method: 'POST',
      }),
      invalidatesTags: ['RecurringBuy', 'Portfolio'],
    }),
  }),
});

export const {
  useListRecurringBuysQuery,
  useCreateRecurringBuyMutation,
  usePauseRecurringBuyMutation,
  useResumeRecurringBuyMutation,
  useDeleteRecurringBuyMutation,
  useListRecurringBuyRunsQuery,
  useCompleteRecurringBuyRunMutation,
} = recurringApi;
