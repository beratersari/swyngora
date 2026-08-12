import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PortfolioPage } from './PortfolioPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListPortfoliosQuery: () => ({
      data: { portfolios: [{ id: 'b1', name: 'Main', currency: 'USDT' }], count: 1 },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetPortfolioQuery: () => ({
      data: {
        id: 'b1',
        name: 'Main',
        currency: 'USDT',
        equity: 10000,
        cashBalance: 10000,
        availableCash: 10000,
        positionsValue: 0,
        unrealizedPnL: 0,
        realizedPnLTotal: 0,
        totalPnL: 0,
        positions: [],
      },
      isLoading: false,
      isError: false,
    }),
    useGetPortfolioPerformanceQuery: () => ({ data: { points: [] }, isLoading: false, isError: false }),
    useListPortfolioOrdersQuery: () => ({ data: { orders: [] }, isLoading: false }),
    useListPortfolioTradesQuery: () => ({ data: { trades: [] }, isLoading: false }),
    useListPortfolioCashMovementsQuery: () => ({ data: { movements: [] }, isLoading: false }),
    useCreatePortfolioMutation: () => [vi.fn(), { isLoading: false }],
    usePlacePortfolioOrderMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useCancelPortfolioOrderMutation: () => [vi.fn(), { isLoading: false }],
    useDepositPortfolioCashMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useWithdrawPortfolioCashMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
  };
});

vi.mock('@/libs/realtime', () => ({
  usePortfolioSubscription: () => ({ connected: false }),
  useRealtimeConnected: () => false,
  usePriceSubscription: () => ({ connected: false }),
}));

describe('PortfolioPage', () => {
  it('renders paper desk summary and trade form', async () => {
    renderWithProviders(<PortfolioPage />, { routerEntries: ['/portfolio'] });
    expect(await screen.findByRole('button', { name: /new book|yeni defter/i })).toBeInTheDocument();
    // Summary strip + market form (buy control is segmented or primary submit)
    expect(screen.getAllByText(/equity|özkaynak/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/place market order|piyasa emri/i)).toBeInTheDocument();
  });
});
