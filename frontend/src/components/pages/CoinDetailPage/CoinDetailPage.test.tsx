import { describe, expect, it, vi, beforeEach, beforeAll } from 'vitest';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import { CoinDetailPage } from './CoinDetailPage';

const mockIntervals = vi.fn();
const mockTicker = vi.fn();
const mockSupply = vi.fn();
const mockCandles = vi.fn();
const mockIndicators = vi.fn();
const mockOrderBook = vi.fn();
const mockOrderHeatmap = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListIntervalsQuery: () => mockIntervals(),
    useGetTicker24hQuery: () => mockTicker(),
    useGetSupplyQuery: () => mockSupply(),
    useGetCandlesQuery: () => mockCandles(),
    useGetIndicatorsQuery: () => mockIndicators(),
    useGetSpotOrderBookQuery: () => mockOrderBook(),
    useGetSpotOrderBookHeatmapQuery: () => mockOrderHeatmap(),
  };
});

vi.mock('@/components/molecules/CandleChartHost', () => ({
  CandleChartHost: (props: { data: unknown[] }) => (
    <div data-testid="candle-chart" data-count={String((props.data ?? []).length)} />
  ),
}));
vi.mock('@/components/molecules/IndicatorChartHost', () => ({
  IndicatorChartHost: () => <div data-testid="indicator-chart" />,
}));

function renderDetail(path: string) {
  return renderWithProviders(
    <Routes>
      <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
    </Routes>,
    { routerEntries: [path] },
  );
}

const candle = {
  openTime: '2024-01-01T00:00:00Z',
  open: '1',
  high: '2',
  low: '0.5',
  close: '1.5',
  volume: '10',
};

describe('CoinDetailPage', () => {
  beforeEach(() => {
    mockIntervals.mockReturnValue({
      data: { intervals: ['1h', '4h'], exchange: 'binance' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockTicker.mockReturnValue({
      data: { lastPrice: '67000', priceChangePercent: '1.5', symbol: 'BTCUSDT' },
      currentData: { lastPrice: '67000', priceChangePercent: '1.5', symbol: 'BTCUSDT' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockSupply.mockReturnValue({
      data: { asset: 'BTC', name: 'Bitcoin', circulatingSupply: 19e6 },
      currentData: { asset: 'BTC', name: 'Bitcoin', circulatingSupply: 19e6 },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockCandles.mockReturnValue({
      data: { candles: [candle] },
      currentData: { candles: [candle] },
      isLoading: false,
      isError: false,
      isSuccess: true,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockIndicators.mockReturnValue({
      data: {
        latest: { rsi: 55, ema: { '12': 100, '26': 99 } },
        points: [{ openTime: '2024-01-01T00:00:00Z', rsi: 55, ema: { '12': 100, '26': 99 } }],
      },
      currentData: {
        latest: { rsi: 55, ema: { '12': 100, '26': 99 } },
        points: [{ openTime: '2024-01-01T00:00:00Z', rsi: 55, ema: { '12': 100, '26': 99 } }],
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockOrderBook.mockReturnValue({
      data: {
        lastPrice: '100',
        groupSize: '0.1',
        suggestedGroupSizes: ['0.01', '0.1'],
        bids: [{ price: '99.9', quantity: '1', cumulative: '1', isWall: false }],
        asks: [{ price: '100.1', quantity: '1', cumulative: '1', isWall: false }],
        spread: '0.2',
      },
      currentData: {
        lastPrice: '100',
        groupSize: '0.1',
        suggestedGroupSizes: ['0.01', '0.1'],
        bids: [{ price: '99.9', quantity: '1', cumulative: '1', isWall: false }],
        asks: [{ price: '100.1', quantity: '1', cumulative: '1', isWall: false }],
        spread: '0.2',
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockOrderHeatmap.mockReturnValue({
      data: {
        symbol: 'BTCUSDT',
        windowSeconds: 600,
        columns: [
          {
            t: '2026-08-16T12:00:00.000Z',
            mid: '100',
            bids: [{ price: '99', notional: '1000' }],
            asks: [{ price: '101', notional: '800' }],
          },
        ],
      },
      currentData: {
        symbol: 'BTCUSDT',
        windowSeconds: 600,
        columns: [
          {
            t: '2026-08-16T12:00:00.000Z',
            mid: '100',
            bids: [{ price: '99', notional: '1000' }],
            asks: [{ price: '101', notional: '800' }],
          },
        ],
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
  });

  it('renders symbol header and chart host', async () => {
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByText('BTC/USDT')).toBeInTheDocument();
    expect(screen.getByTestId('candle-chart')).toBeInTheDocument();
    expect(screen.getByTestId('order-book')).toBeInTheDocument();
    expect(screen.getByTestId('order-depth-chart')).toBeInTheDocument();
    expect(screen.getByTestId('order-heatmap')).toBeInTheDocument();
  });

  it('rejects unknown exchange path segments', async () => {
    renderDetail('/markets/kraken/BTCUSDT');
    expect(await screen.findByText('Unknown exchange')).toBeInTheDocument();
    expect(screen.queryByTestId('candle-chart')).not.toBeInTheDocument();
  });

  it('shows hard candle error when no chart data', async () => {
    mockCandles.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      isSuccess: false,
      isFetching: false,
      error: { status: 502, data: {} },
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByText('Could not load candles')).toBeInTheDocument();
    expect(screen.queryByTestId('candle-chart')).not.toBeInTheDocument();
  });

  it('keeps chart and soft-warns when candle poll fails with prior data', async () => {
    mockCandles.mockReturnValue({
      data: { candles: [candle] },
      currentData: { candles: [candle] },
      isLoading: false,
      isError: true,
      isSuccess: false,
      isFetching: false,
      error: { status: 502, data: {} },
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByTestId('candle-chart')).toBeInTheDocument();
    expect(screen.getByText('Could not load candles')).toBeInTheDocument();
  });

  it('shows empty chart alert when success with zero candles', async () => {
    mockCandles.mockReturnValue({
      data: { candles: [] },
      currentData: { candles: [] },
      isLoading: false,
      isError: false,
      isSuccess: true,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByText('No candle data')).toBeInTheDocument();
  });

  it('shows intervals warning but still loads series on intervals error', async () => {
    mockIntervals.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      isFetching: false,
      error: { status: 500, data: {} },
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByTestId('candle-chart')).toBeInTheDocument();
  });

  it('rewrites unsupported interval in URL', async () => {
    mockIntervals.mockReturnValue({
      data: { intervals: ['15m', '1h'], exchange: 'binance' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT?interval=3m');
    await waitFor(() => {
      expect(screen.getByText('BTC/USDT')).toBeInTheDocument();
    });
  });

  it('shows ticker and supply errors in stats', async () => {
    mockTicker.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      isFetching: false,
      error: { status: 500, data: {} },
      refetch: vi.fn(),
    });
    mockSupply.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      isFetching: false,
      error: { status: 404, data: {} },
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findAllByRole('alert')).not.toHaveLength(0);
  });

  it('shows indicator error message', async () => {
    mockIndicators.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      isFetching: false,
      error: { status: 500, data: {} },
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    // IndicatorPanel error title from i18n
    expect(await screen.findByText(/indicators? unavailable|gösterge/i)).toBeInTheDocument();
  });

  it('refresh button invokes refetch hooks', async () => {
    const user = userEvent.setup();
    const refetchIntervals = vi.fn();
    const refetchTicker = vi.fn();
    const refetchCandles = vi.fn();
    mockIntervals.mockReturnValue({
      data: { intervals: ['1h'], exchange: 'binance' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: refetchIntervals,
    });
    mockTicker.mockReturnValue({
      data: { lastPrice: '1', priceChangePercent: '0', symbol: 'BTCUSDT' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: refetchTicker,
    });
    mockSupply.mockReturnValue({
      data: { asset: 'BTC' },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockCandles.mockReturnValue({
      data: { candles: [candle] },
      currentData: { candles: [candle] },
      isLoading: false,
      isError: false,
      isSuccess: true,
      isFetching: false,
      refetch: refetchCandles,
    });
    mockIndicators.mockReturnValue({
      data: { latest: { rsi: 50 }, points: [] },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    await user.click(await screen.findByRole('button', { name: /refresh/i }));
    expect(refetchIntervals).toHaveBeenCalled();
    expect(refetchTicker).toHaveBeenCalled();
    expect(refetchCandles).toHaveBeenCalled();
  });
});
