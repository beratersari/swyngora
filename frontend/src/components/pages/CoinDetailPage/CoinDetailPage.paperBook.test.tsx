import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import { CoinDetailPage } from './CoinDetailPage';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});

const placePaperOrder = vi.fn();

vi.mock('@/components/molecules/CandleChartHost', () => ({
  CandleChartHost: () => <div data-testid="candle-chart" />,
}));

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  const empty = {
    data: undefined,
    currentData: undefined,
    isLoading: false,
    isError: false,
    isFetching: false,
    isSuccess: false,
    refetch: vi.fn(),
  };
  const candle = {
    openTime: '2024-01-01T00:00:00Z',
    open: '1',
    high: '2',
    low: '0.5',
    close: '1.5',
    volume: '10',
  };
  return {
    ...actual,
    useGetFxRatesQuery: () => ({
      data: { rates: { USD: 1 }, asOf: '2026-08-22' },
      isError: false,
    }),
    useListIntervalsQuery: () => ({
      data: { intervals: ['1h'], exchange: 'binance' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetTicker24hQuery: () => ({
      data: { lastPrice: '67000', priceChangePercent: '1.5', symbol: 'BTCUSDT' },
      currentData: { lastPrice: '67000', priceChangePercent: '1.5', symbol: 'BTCUSDT' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetCandlesQuery: () => ({
      data: { candles: [candle] },
      currentData: { candles: [candle] },
      isLoading: false,
      isError: false,
      isSuccess: true,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetIndicatorsQuery: () => empty,
    useGetSupplyQuery: () => empty,
    useGetHoldersQuery: () => empty,
    useGetAssetProfileQuery: () => empty,
    useGetSpotOrderBookQuery: () => empty,
    useGetSpotOrderBookHeatmapQuery: () => empty,
    useGetOpenInterestQuery: () => empty,
    useGetMarketLiquidationsQuery: () => empty,
    useGetMarketLiquidationHuntHeatmapQuery: () => empty,
    useGetMarketCvdQuery: () => empty,
    useGetPumpEventsQuery: () => empty,
    useGetWatchlistQuery: () => ({ data: { items: [] }, isLoading: false }),
    useListDelistScheduleQuery: () => empty,
    useGetPostDelistQuery: () => empty,
    useListPortfoliosQuery: () => ({
      data: {
        portfolios: [
          { id: 'book-a', name: 'Alpha' },
          { id: 'book-b', name: 'Beta' },
        ],
      },
      isLoading: false,
    }),
    useGetPortfolioQuery: () => empty,
    useListScannerResultsQuery: () => ({ data: { results: [] }, isLoading: false }),
    useLazyGetCandlesQuery: () => [vi.fn()],
    useLazyGetPumpEventsQuery: () => [vi.fn()],
    useAddWatchlistItemMutation: () => [vi.fn(), { isLoading: false }],
    useRemoveWatchlistItemMutation: () => [vi.fn(), { isLoading: false }],
    usePlacePortfolioOrderMutation: () => [
      placePaperOrder,
      { isLoading: false, isError: false },
    ],
  };
});

vi.mock('@/libs/realtime', () => ({
  usePriceSubscription: () => ({ connected: false }),
  usePortfolioSubscription: () => ({ connected: false }),
  useRealtimeConnected: () => false,
}));

describe('CoinDetailPage paper trade without a selected book', () => {
  beforeEach(() => {
    localStorage.removeItem('swyngora.portfolioBookId');
    placePaperOrder.mockReset();
    placePaperOrder.mockReturnValue({ unwrap: vi.fn().mockResolvedValue({}) });
  });

  it('does not place a paper order without portfolioId when 2+ books exist and none is selected', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      </Routes>,
      { routerEntries: ['/markets/binance/BTCUSDT?tab=trade'] },
    );

    const qty = await screen.findByRole('spinbutton', { name: /quantity|miktar/i });
    await user.clear(qty);
    await user.type(qty, '1');
    await user.click(screen.getByRole('button', { name: /paper buy|kağıt al/i }));

    expect(placePaperOrder).not.toHaveBeenCalled();
  });
});
