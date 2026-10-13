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
      currentData: {
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
      isFetching: false,
      isError: false,
    }),
    useGetPortfolioPerformanceQuery: () => ({ data: { points: [] }, isLoading: false, isError: false }),
    useListPortfolioOrdersQuery: () => ({ data: { orders: [] }, isLoading: false }),
    useListPortfolioTradesQuery: () => ({ data: { trades: [] }, isLoading: false }),
    useGetPortfolioSituationLeaveQuery: () => ({
      currentData: {
        counted: 1,
        skipped: 0,
        unreadable: 0,
        rows: [
          {
            clause: 'priceCvd',
            kind: 'price_up_cvd_down',
            still: { count: 1, profitable: 1, realizedPnL: 4, closes: [] },
            left: { count: 0, profitable: 0, realizedPnL: 0, closes: [] },
          },
        ],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
    }),
    useGetPortfolioSituationRecordQuery: () => ({
      currentData: {
        counted: 1,
        skipped: 0,
        rows: [
          {
            clause: 'priceCvd',
            kind: 'price_up_cvd_down',
            count: 1,
            profitable: 1,
            realizedPnL: 4,
          },
        ],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
    }),
    useListPortfolioRebalancesQuery: () => ({ data: { rebalances: [] }, isLoading: false }),
    useGetPortfolioRebalanceQuery: () => ({ data: undefined, isFetching: false }),
    useContinuePortfolioRebalanceMutation: () => [vi.fn(), { isLoading: false }],
    useListPortfolioCashMovementsQuery: () => ({ data: { movements: [] }, isLoading: false }),
    useListTransferAttemptsQuery: () => ({ data: { attempts: [] }, isLoading: false }),
    useTransferPortfolioCashMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useCreatePortfolioMutation: () => [vi.fn(), { isLoading: false }],
    usePlacePortfolioOrderMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useAmendPortfolioOrderMutation: () => [vi.fn(), { isLoading: false }],
    useCancelPortfolioOrderMutation: () => [vi.fn(), { isLoading: false }],
    useDepositPortfolioCashMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useWithdrawPortfolioCashMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useListMarginPositionsQuery: () => ({ data: { positions: [] }, isLoading: false }),
    useSetMarginModeMutation: () => [vi.fn(), { isLoading: false }],
    usePlaceMarginOrderMutation: () => [vi.fn(), { isLoading: false, isError: false }],
    useCloseMarginPositionMutation: () => [vi.fn(), { isLoading: false }],
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
    expect(screen.getByRole('region', { name: /what your closes made|kapanışların ne getirdiği/i })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: /did leaving pay|ayrılınca ne oldu/i })).toBeInTheDocument();
    // Summary strip + market form (buy control is segmented or primary submit)
    expect(screen.getAllByText(/equity|özkaynak/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/spot order ticket|spot emir fişi|order ticket|emir fişi/i)).toBeInTheDocument();
    expect(screen.getByText(/margin ticket|marjin fişi/i)).toBeInTheDocument();
  });
});
