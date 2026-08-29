import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import { CoinDetailPage } from './CoinDetailPage';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});

const candleArgs: unknown[] = [];

vi.mock('@/components/molecules/CandleChartHost', () => ({
  CandleChartHost: (props: { data: unknown[] }) => (
    <div data-testid="candle-chart" data-count={String((props.data ?? []).length)} />
  ),
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
    openTime: '2026-08-16T22:00:00Z',
    open: '0.11',
    high: '0.12',
    low: '0.10',
    close: '0.11',
    volume: '10',
  };
  return {
    ...actual,
    useGetFxRatesQuery: () => ({
      data: { rates: { USD: 1 }, asOf: '2026-08-22' },
      isError: false,
    }),
    useListIntervalsQuery: () => ({
      data: { intervals: ['1h'], exchange: 'bybit' },
      currentData: { intervals: ['1h'], exchange: 'bybit' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetTicker24hQuery: () => ({
      data: { lastPrice: '0.50', priceChangePercent: '1.2', symbol: 'VICUSDT', halted: false },
      currentData: { lastPrice: '0.50', priceChangePercent: '1.2', symbol: 'VICUSDT', halted: false },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetCandlesQuery: (arg: unknown) => {
      candleArgs.push(arg);
      return {
        data: { candles: [candle] },
        currentData: { candles: [candle] },
        isLoading: false,
        isError: false,
        isSuccess: true,
        isFetching: false,
        refetch: vi.fn(),
      };
    },
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
    // RTK arg-change: previous venue's schedule is still in `.data` while
    // the current venue's request has no currentData yet.
    useListDelistScheduleQuery: () => ({
      data: {
        exchange: 'binance',
        enabled: true,
        items: [{ symbol: 'VICUSDT', delistTime: '2026-08-17T00:00:00Z' }],
      },
      currentData: undefined,
      isLoading: false,
      isFetching: true,
      isError: false,
    }),
    useGetPostDelistQuery: () => ({
      data: {
        available: true,
        source: 'coingecko',
        sourceLabel: 'CoinGecko',
        lastPrice: '0.1234',
        quote: 'USD',
      },
      currentData: {
        available: true,
        source: 'coingecko',
        sourceLabel: 'CoinGecko',
        lastPrice: '0.1234',
        quote: 'USD',
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      isSuccess: true,
      refetch: vi.fn(),
    }),
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

describe('CoinDetailPage stale delist schedule across venues', () => {
  beforeEach(() => {
    candleArgs.length = 0;
    vi.useFakeTimers({ now: new Date('2026-08-22T12:00:00Z') });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not treat Binance delist of VICUSDT as a Bybit halt while Bybit schedule is in flight', () => {
    renderWithProviders(
      <Routes>
        <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      </Routes>,
      { routerEntries: ['/markets/bybit/VICUSDT'] },
    );

    expect(screen.queryByTestId('post-delist-panel')).not.toBeInTheDocument();
    expect(screen.queryByText('After delist')).not.toBeInTheDocument();

    const withEndTime = candleArgs.filter(
      (arg): arg is { endTime?: string } =>
        Boolean(arg && typeof arg === 'object' && 'endTime' in (arg as object)),
    );
    expect(withEndTime).toEqual([]);
  });
});
