import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { PortfolioPage } from './PortfolioPage';

const useGetPortfolioQuery = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListPortfoliosQuery: () => ({
      data: {
        portfolios: [
          { id: 'book-a', name: 'Alpha', currency: 'USDT' },
          { id: 'book-b', name: 'Beta', currency: 'USDT' },
        ],
        count: 2,
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetPortfolioQuery: (...args: unknown[]) => useGetPortfolioQuery(...args),
    useGetPortfolioPerformanceQuery: () => ({ data: { points: [] }, isLoading: false, isError: false }),
    useListPortfolioOrdersQuery: () => ({ data: { orders: [] }, isLoading: false }),
    useListPortfolioTradesQuery: () => ({ data: { trades: [] }, isLoading: false }),
    useListPortfolioCashMovementsQuery: () => ({ data: { movements: [] }, isLoading: false }),
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

describe('PortfolioPage multi-book query args', () => {
  beforeEach(() => {
    localStorage.removeItem('swyngora.portfolioBookId');
    useGetPortfolioQuery.mockReset();
    useGetPortfolioQuery.mockReturnValue({
      currentData: undefined,
      isLoading: false,
      isFetching: false,
      isError: false,
    });
  });

  it('does not fetch a book-less snapshot when 2+ books exist and none is selected', () => {
    renderWithProviders(<PortfolioPage />, { routerEntries: ['/portfolio'] });

    expect(useGetPortfolioQuery).toHaveBeenCalled();
    const [arg, opts] = useGetPortfolioQuery.mock.calls[0] as [
      { portfolioId?: string } | undefined,
      { skip?: boolean } | undefined,
    ];
    // OpenAPI: portfolioId is required when more than one paper book exists.
    // Firing GET /portfolio with neither skip nor a book id 400s the desk.
    const skipped = Boolean(opts?.skip);
    const targeted = Boolean(arg && typeof arg === 'object' && arg.portfolioId);
    expect(skipped || targeted).toBe(true);
  });
});
