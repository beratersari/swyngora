import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import { MARKETS_RETURN_STORAGE_KEY } from '@/libs/utils';
import { CoinDetailPage } from './CoinDetailPage';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});

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
  return {
    ...actual,
    useListIntervalsQuery: () => ({
      data: { intervals: ['1h'], exchange: 'coinbase' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetTicker24hQuery: () => ({
      data: { lastPrice: '100', priceChangePercent: '1', symbol: 'BTC-USD' },
      currentData: { lastPrice: '100', priceChangePercent: '1', symbol: 'BTC-USD' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetCandlesQuery: () => empty,
    useGetIndicatorsQuery: () => empty,
    useGetSupplyQuery: () => empty,
    useGetHoldersQuery: () => empty,
    useGetAssetProfileQuery: () => empty,
    useGetSpotOrderBookQuery: () => empty,
    useGetSpotOrderBookHeatmapQuery: () => empty,
    useGetOpenInterestQuery: () => empty,
    useGetMarketLiquidationsQuery: () => empty,
    useGetMarketCvdQuery: () => empty,
    useGetPumpEventsQuery: () => empty,
    useGetWatchlistQuery: () => ({ data: { items: [] }, isLoading: false }),
    useListDelistScheduleQuery: () => empty,
    useGetPostDelistQuery: () => empty,
    useListPortfoliosQuery: () => ({ data: { portfolios: [] }, isLoading: false }),
    useGetPortfolioQuery: () => empty,
    useListScannerResultsQuery: () => ({ data: { results: [] }, isLoading: false }),
    useLazyGetCandlesQuery: () => [vi.fn()],
    useLazyGetPumpEventsQuery: () => [vi.fn()],
    useAddWatchlistItemMutation: () => [vi.fn(), { isLoading: false }],
    useRemoveWatchlistItemMutation: () => [vi.fn(), { isLoading: false }],
    usePlacePortfolioOrderMutation: () => [vi.fn(), { isLoading: false, isError: false }],
  };
});

vi.mock('@/libs/realtime', () => ({
  usePriceSubscription: () => ({ connected: false }),
  usePortfolioSubscription: () => ({ connected: false }),
  useRealtimeConnected: () => false,
}));

describe('CoinDetailPage back-to-markets', () => {
  beforeEach(() => {
    sessionStorage.setItem(
      MARKETS_RETURN_STORAGE_KEY,
      '?exchange=coinbase&quote=USD&q=eth&tag=defi&offset=50',
    );
  });

  it('restores the markets list filters the user came from', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      </Routes>,
      { routerEntries: ['/markets/coinbase/BTC-USD'] },
    );

    const back = await screen.findByRole('link', { name: /markets|piyasalar/i });
    const href = back.getAttribute('href') ?? '';
    expect(href).toContain('exchange=coinbase');
    expect(href).toContain('q=eth');
    expect(href).toContain('offset=50');
  });
});
