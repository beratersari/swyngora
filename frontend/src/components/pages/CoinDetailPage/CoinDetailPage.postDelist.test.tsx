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

vi.mock('@/components/molecules/CandleChartHost', () => ({
  CandleChartHost: (props: { data: unknown[]; seriesKey?: string }) => (
    <div
      data-testid={props.seriesKey?.startsWith('post-delist:') ? 'post-delist-chart' : 'candle-chart'}
      data-count={String((props.data ?? []).length)}
    />
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
      data: { intervals: ['1h'], exchange: 'binance' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetTicker24hQuery: () => ({
      data: { lastPrice: '0.11', priceChangePercent: '-2', symbol: 'VICUSDT', halted: true },
      currentData: { lastPrice: '0.11', priceChangePercent: '-2', symbol: 'VICUSDT', halted: true },
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
    useListDelistScheduleQuery: () => ({
      data: {
        exchange: 'binance',
        enabled: true,
        items: [{ symbol: 'VICUSDT', delistTime: '2026-08-17T00:00:00Z' }],
      },
      isLoading: false,
      isError: false,
    }),
    useGetPostDelistQuery: () => ({
      data: {
        available: true,
        source: 'coingecko',
        sourceLabel: 'CoinGecko',
        note: 'This venue no longer trades this pair.',
        lastPrice: '0.1234',
        priceChangePercent: '-3.5',
        quote: 'USD',
        interval: '1d',
        candles: [
          {
            openTime: '2026-08-20T00:00:00Z',
            open: '0.10',
            high: '0.14',
            low: '0.09',
            close: '0.1234',
            volume: '0',
          },
        ],
      },
      currentData: {
        available: true,
        source: 'coingecko',
        sourceLabel: 'CoinGecko',
        lastPrice: '0.1234',
        priceChangePercent: '-3.5',
        quote: 'USD',
        interval: '1d',
        candles: [
          {
            openTime: '2026-08-20T00:00:00Z',
            open: '0.10',
            high: '0.14',
            low: '0.09',
            close: '0.1234',
            volume: '0',
          },
        ],
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

describe('CoinDetailPage post-delist panel', () => {
  beforeEach(() => {
    vi.useFakeTimers({ now: new Date('2026-08-22T12:00:00Z') });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows CoinGecko movement after Binance halt', () => {
    renderWithProviders(
      <Routes>
        <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      </Routes>,
      { routerEntries: ['/markets/binance/VICUSDT'] },
    );
    expect(screen.getByTestId('post-delist-panel')).toBeInTheDocument();
    expect(screen.getByText('After delist')).toBeInTheDocument();
    expect(screen.getByText('0.1234')).toBeInTheDocument();
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-count', '2');
    expect(screen.queryByText('Off-venue movement')).not.toBeInTheDocument();
  });
});
