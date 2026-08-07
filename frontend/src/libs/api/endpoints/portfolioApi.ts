import { baseApi } from '../baseApi';
import type { components } from '../generated/schema';

export type PortfolioView = components['schemas']['PortfolioView'];
export type PortfolioPerformance = components['schemas']['PortfolioPerformance'];
export type PortfolioEquityPoint = components['schemas']['PortfolioEquityPoint'];
export type PortfolioPerformancePeriod = NonNullable<
  NonNullable<components['schemas']['PortfolioPerformance']['period']>
>;

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
  }),
});

export const { useGetPortfolioQuery, useGetPortfolioPerformanceQuery } = portfolioApi;
