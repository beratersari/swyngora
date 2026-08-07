import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type PortfolioView = components['schemas']['PortfolioView'];
export type PortfolioPerformance = components['schemas']['PortfolioPerformance'];
export type PortfolioEquityPoint = components['schemas']['PortfolioEquityPoint'];
export type PortfolioPerformancePeriod = NonNullable<
  NonNullable<components['schemas']['PortfolioPerformance']['period']>
>;
export type PortfolioCashMovement = components['schemas']['PortfolioCashMovement'];
export type PortfolioCashMoveResponse = components['schemas']['PortfolioCashMoveResponse'];

type CashMoveArg = { amount: number; note?: string };
type CashMovementListResponse = {
  clientId?: string;
  movements?: PortfolioCashMovement[];
  count?: number;
  total?: number;
  limit?: number;
  offset?: number;
};

export const portfolioApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getPortfolio: build.query<PortfolioView, void>({
      query: () => '/api/v1/portfolio',
      providesTags: ['Portfolio'],
    }),
    getPortfolioPerformance: build.query<PortfolioPerformance, { period?: PortfolioPerformancePeriod } | void>({
      query: (arg) => ({
        url: '/api/v1/portfolio/performance',
        params: { period: arg?.period ?? '1w' },
      }),
      providesTags: ['Portfolio'],
    }),
    listPortfolioCashMovements: build.query<CashMovementListResponse, { limit?: number; offset?: number } | void>({
      query: (arg) => ({
        url: '/api/v1/portfolio/cash-movements',
        params: { limit: arg?.limit ?? 50, offset: arg?.offset ?? 0 },
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
  useGetPortfolioQuery,
  useGetPortfolioPerformanceQuery,
  useListPortfolioCashMovementsQuery,
  useDepositPortfolioCashMutation,
  useWithdrawPortfolioCashMutation,
} = portfolioApi;
