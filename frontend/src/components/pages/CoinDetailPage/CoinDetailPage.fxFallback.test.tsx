import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import { DISPLAY_CURRENCY_STORAGE_KEY } from '@/libs/utils';
import { CoinDetailPage } from './CoinDetailPage';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});

const lastChartData: { close: number; open: number }[] = [];

vi.mock('@/components/molecules/CandleChartHost', () => ({
  CandleChartHost: (props: { data: { close: number; open: number }[] }) => {
    lastChartData.splice(0, lastChartData.length, ...(props.data ?? []));
    return <div data-testid="candle-chart" data-close={String(props.data?.[0]?.close ?? '')} />;
  },
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
    open: '0.035',
    high: '0.036',
    low: '0.034',
    close: '0.035',
    volume: '10',
  };
  return {
    ...actual,
    useGetFxRatesQuery: () => ({
      data: { rates: { TRY: 40, EUR: 0.9, USD: 1 }, asOf: '2026-01-01' },
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
      data: { lastPrice: '0.035', priceChangePercent: '1.5', symbol: 'ETHBTC' },
      currentData: { lastPrice: '0.035', priceChangePercent: '1.5', symbol: 'ETHBTC' },
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

describe('CoinDetailPage display-currency chart fallback', () => {
  beforeEach(() => {
    lastChartData.length = 0;
    localStorage.setItem(DISPLAY_CURRENCY_STORAGE_KEY, 'USD');
  });

  it('does not plot native BTC prices on a USD chart when FX cannot convert the quote', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      </Routes>,
      { routerEntries: ['/markets/binance/ETHBTC'] },
    );

    const headerLast = await screen.findByRole('heading', { level: 3 });
    expect(headerLast.textContent?.trim()).toBe('—');

    // Unconverted BTC quotes are not drawn as USD — no native 0.035 series.
    expect(lastChartData[0]?.close).not.toBe(0.035);
    expect(screen.queryByTestId('candle-chart')).toBeNull();
  });
});
