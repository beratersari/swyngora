import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { PortfolioPage } from './PortfolioPage';

const placeOrder = vi.fn();
const deposit = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetFxRatesQuery: () => ({
      data: { rates: { USD: 1 }, asOf: '2026-08-22' },
      isError: false,
    }),
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
    useGetPortfolioQuery: () => ({
      currentData: undefined,
      isLoading: false,
      isFetching: false,
      isError: false,
    }),
    useGetPortfolioPerformanceQuery: () => ({ data: { points: [] }, isLoading: false, isError: false }),
    useListPortfolioOrdersQuery: () => ({ data: { orders: [] }, isLoading: false }),
    useListPortfolioTradesQuery: () => ({ data: { trades: [] }, isLoading: false }),
    useListPortfolioCashMovementsQuery: () => ({ data: { movements: [] }, isLoading: false }),
    useCreatePortfolioMutation: () => [vi.fn(), { isLoading: false }],
    usePlacePortfolioOrderMutation: () => [
      placeOrder,
      { isLoading: false, isError: false },
    ],
    useAmendPortfolioOrderMutation: () => [vi.fn(), { isLoading: false }],
    useCancelPortfolioOrderMutation: () => [vi.fn(), { isLoading: false }],
    useDepositPortfolioCashMutation: () => [deposit, { isLoading: false, isError: false }],
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

describe('PortfolioPage unselected book mutations', () => {
  beforeEach(() => {
    localStorage.removeItem('swyngora.portfolioBookId');
    placeOrder.mockReset();
    deposit.mockReset();
    placeOrder.mockReturnValue({ unwrap: vi.fn().mockResolvedValue({}) });
    deposit.mockReturnValue({ unwrap: vi.fn().mockResolvedValue({}) });
  });

  it('does not place a spot order without portfolioId when 2+ books exist and none is selected', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PortfolioPage />, { routerEntries: ['/portfolio'] });

    const symbol = screen.getAllByRole('combobox', { name: /symbol/i })[0];
    await user.click(symbol!);
    await user.keyboard('BTCUSDT');

    const qty = screen.getAllByRole('spinbutton', { name: /quantity|miktar/i })[0];
    await user.clear(qty!);
    await user.type(qty!, '1');

    await user.click(screen.getByRole('button', { name: /paper buy|kağıt al/i }));

    expect(placeOrder).not.toHaveBeenCalled();
  });

  it('does not deposit cash without portfolioId when 2+ books exist and none is selected', async () => {
    const user = userEvent.setup();
    renderWithProviders(<PortfolioPage />, { routerEntries: ['/portfolio'] });

    const amount = screen.getByRole('spinbutton', { name: /amount|tutar/i });
    await user.clear(amount);
    await user.type(amount, '100');
    await user.click(screen.getByRole('button', { name: /deposit|yatır/i }));

    expect(deposit).not.toHaveBeenCalled();
  });
});
