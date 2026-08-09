import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type PortfolioView = components['schemas']['PortfolioView'];
export type PortfolioSummary = components['schemas']['PortfolioSummary'];
export type PortfolioPerformance = components['schemas']['PortfolioPerformance'];
export type PortfolioEquityPoint = components['schemas']['PortfolioEquityPoint'];
export type PortfolioPerformancePeriod = NonNullable<
  NonNullable<components['schemas']['PortfolioPerformance']['period']>
>;
export type PortfolioCashMovement = components['schemas']['PortfolioCashMovement'];
export type PortfolioCashMoveResponse = components['schemas']['PortfolioCashMoveResponse'];

type BookArg = { portfolioId?: string };
type CashMoveArg = { amount: number; note?: string; portfolioId?: string };
type CashMovementListResponse = {
  clientId?: string;
  movements?: PortfolioCashMovement[];
  count?: number;
  total?: number;
  limit?: number;
  offset?: number;
};
type PortfolioListResponse = {
  clientId?: string;
  count?: number;
  portfolios?: PortfolioSummary[];
};

const bookParams = (portfolioId?: string) => (portfolioId ? { portfolioId } : undefined);

export const portfolioApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listPortfolios: build.query<PortfolioListResponse, void>({
      query: () => '/api/v1/portfolios',
      providesTags: ['Portfolio'],
    }),
    getPortfolio: build.query<PortfolioView, BookArg | void>({
      query: (arg) => ({
        url: '/api/v1/portfolio',
        params: bookParams(arg?.portfolioId),
      }),
      providesTags: ['Portfolio'],
    }),
    createPortfolio: build.mutation<PortfolioView, { startingBalance: number; currency?: string; name?: string }>({
      query: (body) => ({ url: '/api/v1/portfolio', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
    renamePortfolio: build.mutation<PortfolioSummary, { id: string; name: string }>({
      query: ({ id, name }) => ({ url: `/api/v1/portfolios/${id}`, method: 'PATCH', body: { name } }),
      invalidatesTags: ['Portfolio'],
    }),
    deletePortfolio: build.mutation<{ deleted?: boolean; id?: string }, string>({
      query: (id) => ({ url: `/api/v1/portfolios/${id}`, method: 'DELETE' }),
      invalidatesTags: ['Portfolio'],
    }),
    getPortfolioPerformance: build.query<
      PortfolioPerformance,
      ({ period?: PortfolioPerformancePeriod } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/performance',
        params: { period: arg?.period ?? '1w', ...bookParams(arg?.portfolioId) },
      }),
      providesTags: ['Portfolio'],
    }),
    listPortfolioCashMovements: build.query<
      CashMovementListResponse,
      ({ limit?: number; offset?: number } & BookArg) | void
    >({
      query: (arg) => ({
        url: '/api/v1/portfolio/cash-movements',
        params: { limit: arg?.limit ?? 50, offset: arg?.offset ?? 0, ...bookParams(arg?.portfolioId) },
      }),
      providesTags: ['Portfolio'],
    }),
    depositPortfolioCash: build.mutation<PortfolioCashMoveResponse, CashMoveArg>({
      query: (body) => ({ url: '/api/v1/portfolio/deposits', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
    withdrawPortfolioCash: build.mutation<PortfolioCashMoveResponse, CashMoveArg>({
      query: (body) => ({ url: '/api/v1/portfolio/withdrawals', method: 'POST', body }),
      invalidatesTags: ['Portfolio'],
    }),
  }),
});

export const {
  useListPortfoliosQuery,
  useGetPortfolioQuery,
  useCreatePortfolioMutation,
  useRenamePortfolioMutation,
  useDeletePortfolioMutation,
  useGetPortfolioPerformanceQuery,
  useListPortfolioCashMovementsQuery,
  useDepositPortfolioCashMutation,
  useWithdrawPortfolioCashMutation,
} = portfolioApi;
