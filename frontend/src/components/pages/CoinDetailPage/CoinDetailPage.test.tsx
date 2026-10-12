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
const mockHolders = vi.fn();
const mockCandles = vi.fn();
const mockIndicators = vi.fn();
const mockOrderBook = vi.fn();
const mockOrderHeatmap = vi.fn();
const mockOpenInterest = vi.fn();
const mockLiquidations = vi.fn();
const mockCvd = vi.fn();
const mockPositioningAnchor = vi.fn();
const mockLiqHeatmap = vi.fn();
const mockProfile = vi.fn();
const mockCandleFootprint = vi.fn();
const mockAbsorption = vi.fn();
const mockPerpDepth = vi.fn();
const mockPerpBook = vi.fn();
const mockPrintSize = vi.fn();
const mockVolumeProfile = vi.fn();
const mockSweeps = vi.fn();
const mockBots = vi.fn();
const mockForensics = vi.fn();
const mockFillPocket = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListIntervalsQuery: () => mockIntervals(),
    useGetTicker24hQuery: () => mockTicker(),
    useGetSupplyQuery: () => mockSupply(),
    useGetHoldersQuery: () => mockHolders(),
    useGetCandlesQuery: () => mockCandles(),
    useGetIndicatorsQuery: () => mockIndicators(),
    useGetSpotOrderBookQuery: () => mockOrderBook(),
    useGetSpotOrderBookHeatmapQuery: () => mockOrderHeatmap(),
    useGetBookGapsQuery: () => ({
      data: {
        easier: 'up',
        upNotional: '1000',
        downNotional: '9000',
        upGapPct: 30,
        downGapPct: 5,
        gaps: [{ direction: 'up', fromPrice: '100', toPrice: '101', widthPct: 0.5, notional: '10' }],
      },
      isError: false,
      isFetching: false,
    }),
    useGetOpenInterestQuery: () => mockOpenInterest(),
    useGetMarketLiquidationsQuery: () => mockLiquidations(),
    useGetMarketLiquidationHuntHeatmapQuery: () => mockLiqHeatmap(),
    useGetMarketLiquidationPrintHeatmapQuery: () => mockLiqHeatmap(),
    useGetMarketLiquidationOpenLevelsQuery: () => ({
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetMarketCvdQuery: () => mockCvd(),
    useGetFillPocketQuery: () => mockFillPocket(),
    useGetMarketSituationQuery: () => ({
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    }),
    useGetMarketForensicsQuery: (...args: unknown[]) => mockForensics(...args),
    useGetMarketPositioningAnchorQuery: () => mockPositioningAnchor(),
    useGetMarketFlowAtPricesQuery: () => ({ data: undefined, isError: false, isFetching: false, isLoading: false, isSuccess: false, refetch: vi.fn() }),
    useGetMarketCandleFootprintQuery: (...args: unknown[]) => mockCandleFootprint(...args),
    useGetMarketLongShortRatioQuery: () => ({ data: undefined, isError: false, isFetching: false, isLoading: false, isSuccess: false, refetch: vi.fn() }),
    useGetMarketHistorySpanQuery: () => ({
      data: { symbol: 'BTCUSDT', series: [{ kind: 'futures_oi', exchange: 'binance', spanSec: 3600 }] },
      currentData: { symbol: 'BTCUSDT', series: [{ kind: 'futures_oi', exchange: 'binance', spanSec: 3600 }] },
      isError: false,
      isFetching: false,
      isLoading: false,
      isSuccess: true,
      refetch: vi.fn(),
    }),
    useGetMarketUnusualQuery: () => ({
      data: undefined,
      currentData: undefined,
      isError: false,
      isFetching: false,
      isLoading: false,
    }),
    useGetMarketLiquidationLevelsQuery: () => ({ data: undefined, isError: false, isFetching: false }),
    useGetMarketPrintSizeQuery: () => mockPrintSize(),
    useGetMarketAbsorptionQuery: () => mockAbsorption(),
    useGetMarketPerpDepthQuery: () => mockPerpDepth(),
    useGetMarketPerpBookHistoryQuery: () => mockPerpBook(),
    useGetMarketBotTrackerQuery: () => mockBots(),
    useGetMarketVolumeProfileQuery: () => mockVolumeProfile(),
    useGetMarketLiquiditySweepsQuery: () => mockSweeps(),
    useGetMarketOrderCountQuery: () => ({ data: undefined, isError: false, isFetching: false }),
    useGetMarketPerpDepthOverviewQuery: () => ({ data: undefined, isError: false, isFetching: false }),
    useGetMarketPressureQuery: () => ({ data: undefined, isError: false, isFetching: false }),
    useGetMarketSnapshotQuery: () => ({ data: undefined, isError: false, isFetching: false }),
    useGetAssetProfileQuery: () => mockProfile(),
  };
});

vi.mock('./CoinFlowTab', () => ({
  CoinFlowTab: () => <div data-testid="coin-flow-tab" />,
}));
vi.mock('./CoinStructureTab', () => ({
  CoinStructureTab: () => <div data-testid="coin-structure-tab" />,
}));

let chartClickAt = 0;

vi.mock('@/components/molecules/CandleChartHost', () => ({
  CandleChartHost: (props: {
    data: unknown[];
    liqHeat?: unknown[];
    candleFootprints?: unknown[];
    footprint?: unknown[];
    priceLines?: { title?: string }[];
    markers?: { text?: string }[];
    vertLines?: { label?: string }[];
    onBarTimeClick?: (timeSec: number) => void;
  }) => (
    <div
      data-testid="candle-chart"
      data-count={String((props.data ?? []).length)}
      data-liq={String(props.liqHeat?.length ?? 0)}
      data-candle-foot={String(props.candleFootprints?.length ?? 0)}
      data-foot-strip={String(props.footprint?.length ?? 0)}
      data-prices={(props.priceLines ?? []).map((line) => line.title).join('|')}
      data-marks={(props.markers ?? []).map((mark) => mark.text).join('|')}
      data-verts={(props.vertLines ?? []).map((line) => line.label).join('|')}
    >
      <button type="button" data-testid="chart-bar-click" onClick={() => props.onBarTimeClick?.(chartClickAt)}>
        candle
      </button>
    </div>
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
    mockFillPocket.mockReset();
    mockFillPocket.mockReturnValue({
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockForensics.mockReset();
    mockForensics.mockReturnValue({
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
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
    mockHolders.mockReturnValue({
      data: {
        asset: 'BTC',
        holderCount: 50_000_000,
        topTenSharePct: 5.4,
        topHolders: [{ address: '34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo', balance: 1, sharePct: 1.18 }],
      },
      currentData: {
        asset: 'BTC',
        holderCount: 50_000_000,
        topTenSharePct: 5.4,
        topHolders: [{ address: '34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo', balance: 1, sharePct: 1.18 }],
      },
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
    const emptyTape = {
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    };
    mockOpenInterest.mockReturnValue(emptyTape);
    mockLiquidations.mockReturnValue(emptyTape);
    mockCvd.mockReturnValue(emptyTape);
    mockPositioningAnchor.mockReturnValue(emptyTape);
    mockLiqHeatmap.mockReturnValue({
      data: {
        symbol: 'BTCUSDT',
        range: '24h',
        prices: [110_000, 100_000],
        times: ['2026-08-29T12:00:00Z', '2026-08-29T12:30:00Z'],
        priceStep: 10_000,
        stepSec: 1800,
        leverageMix: [{ leverage: 10, weight: 0.25 }],
        missingVenues: ['okx'],
        combined: {
          exchange: 'combined',
          totals: [
            [1, 2],
            [3, 4],
          ],
          maxIntensity: 4,
          coverage: 1,
        },
      },
      currentData: {
        symbol: 'BTCUSDT',
        range: '24h',
        prices: [110_000, 100_000],
        times: ['2026-08-29T12:00:00Z', '2026-08-29T12:30:00Z'],
        priceStep: 10_000,
        stepSec: 1800,
        leverageMix: [{ leverage: 10, weight: 0.25 }],
        missingVenues: ['okx'],
        combined: {
          exchange: 'combined',
          totals: [
            [1, 2],
            [3, 4],
          ],
          maxIntensity: 4,
          coverage: 1,
        },
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockProfile.mockReturnValue(emptyTape);
    mockCandleFootprint.mockReturnValue({
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isError: false,
      isFetching: false,
      isSuccess: false,
      refetch: vi.fn(),
    });
    const emptyDesk = {
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    };
    mockAbsorption.mockReturnValue(emptyDesk);
    mockPerpDepth.mockReturnValue(emptyDesk);
    mockPerpBook.mockReturnValue({
      data: { columns: [], gaps: [], stepSec: 60 },
      currentData: { columns: [], gaps: [], stepSec: 60 },
      isError: false,
      isFetching: false,
      isLoading: false,
      refetch: vi.fn(),
    });
    mockPrintSize.mockReturnValue(emptyDesk);
    mockVolumeProfile.mockReturnValue(emptyDesk);
    mockSweeps.mockReturnValue(emptyDesk);
    mockBots.mockReturnValue(emptyDesk);
  });

  it('draws the value area, sweep arrows, and repeated-size squares on the overview chart', async () => {
    const profile = {
      definite: false,
      missing: ['okx'],
      venues: [
        {
          exchange: 'binance',
          poc: { price: '100', high: '102' },
          valueArea: { low: '90', high: '110' },
        },
      ],
    };
    mockVolumeProfile.mockReturnValue({
      data: profile,
      currentData: profile,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    const sweeps = {
      venues: [
        {
          exchange: 'binance',
          sweeps: [{ side: 'high', piercedAt: '2024-01-01T00:00:00Z' }],
        },
      ],
    };
    mockSweeps.mockReturnValue({
      data: sweeps,
      currentData: sweeps,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    const bots = {
      clusters: [
        { side: 'buy', count: 4, from: '2024-01-01T00:00:00Z', to: '2024-01-01T00:00:30Z' },
      ],
    };
    mockBots.mockReturnValue({
      data: bots,
      currentData: bots,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    const chart = await screen.findByTestId('candle-chart');
    expect(chart).toHaveAttribute('data-prices', expect.stringContaining('POC'));
    expect(chart.getAttribute('data-prices')).toContain('VAH');
    expect(chart.getAttribute('data-prices')).toContain('VAL');
    expect(chart.getAttribute('data-marks')).toContain('Sweep high');
    expect(chart.getAttribute('data-marks')).toContain('4× buy');
    expect(screen.getByTestId('chart-profile-venue')).toHaveTextContent(/OKX/);
    expect(screen.getByTestId('chart-sweeps')).toHaveTextContent(/Binance/);
    expect(screen.getByTestId('chart-bots')).toBeInTheDocument();
  });

  it('marks the paper fill and the nearer pocket on the chart and the trade tab', async () => {
    const user = userEvent.setup();
    const mark = {
      open: true,
      fillPrice: 100,
      pocketSide: 'long' as const,
      pocketPrice: 99,
      gapPct: 1,
      pocketFromFill: 'below' as const,
      marginLiqPrice: 80.5,
      marginGapPct: 10.6,
      marginFromPocket: 'below' as const,
    };
    mockFillPocket.mockReturnValue({
      data: mark,
      currentData: mark,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    const chart = await screen.findByTestId('candle-chart');
    expect(chart.getAttribute('data-prices')).toContain('Paper fill');
    expect(chart.getAttribute('data-prices')).toContain('Long pocket · 1% below');
    expect(chart.getAttribute('data-prices')).toContain('Paper liquidation');
    expect(screen.getByText(/Paper fill at 100/)).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: 'Trade' }));
    expect(screen.getAllByText(/Paper liquidation is 10.6% below that pocket/).length).toBeGreaterThan(0);
  });

  it('renders symbol header and chart host', async () => {
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByText('BTC/USDT')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Overview' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Holders' })).toBeInTheDocument();
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-liq', '4');
    expect(screen.getByTestId('chart-liq-mix')).toHaveTextContent(/10× 25%/);
    expect(screen.getByTestId('chart-liq-missing')).toHaveTextContent(/OKX/);
    expect(screen.getByRole('switch', { name: /liq map/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /open levels/i })).toBeChecked();
    expect(screen.queryByTestId('chart-open-withheld')).not.toBeInTheDocument();
    expect(screen.getByTestId('chart-recorded')).toHaveTextContent(/binance 1h/i);
    expect(screen.queryByTestId('order-book')).not.toBeInTheDocument();
    expect(screen.queryByTestId('chart-candle-footprint')).not.toBeInTheDocument();
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-candle-foot', '0');
    expect(screen.getByRole('radio', { name: 'All' })).toBeChecked();
  });

  /** Ant Design desk radios ignore pointer events. Click the visible label. */
  async function chooseDesk(user: ReturnType<typeof userEvent.setup>, name: string) {
    const radio = screen.getByRole('radio', { name });
    const label = radio.closest('label');
    if (!label) throw new Error(`desk option ${name} has no label`);
    await user.click(label);
  }

  it('focuses positioning, order flow, and liquidity without treating a missing venue as zero', async () => {
    const user = userEvent.setup();
    const absorption = {
      combined: { complete: false, summary: 'partial' },
      venues: [
        { exchange: 'binance', complete: true, summary: 'bids absorbing sells' },
        { exchange: 'okx', complete: false, summary: 'short' },
      ],
    };
    mockAbsorption.mockReturnValue({
      data: absorption,
      currentData: absorption,
      isError: false,
      isFetching: false,
      isLoading: false,
      refetch: vi.fn(),
    });
    const prints = {
      combined: { windows: [{ window: '15m', complete: false, summary: 'combined off' }] },
      venues: [
        {
          exchange: 'binance',
          windows: [{ window: '15m', complete: true, summary: 'Last 15m: buy prints averaged 2K USDT.' }],
        },
        { exchange: 'bybit', error: 'quiet' },
      ],
    };
    mockPrintSize.mockReturnValue({
      data: prints,
      currentData: prints,
      isError: false,
      isFetching: false,
      isLoading: false,
      refetch: vi.fn(),
    });
    const depth = {
      rangePct: '1',
      venues: [
        { exchange: 'binance', headline: { rangePct: '1', complete: true, below: { notional: '60' }, above: { notional: '40' } } },
        { exchange: 'bybit', error: 'stale' },
      ],
      combined: {
        exchange: 'all',
        headline: { rangePct: '1', complete: true, below: { notional: '60' }, above: { notional: '40' } },
      },
    };
    const book = {
      stepSec: 60,
      columns: [
        {
          time: '2024-01-01T00:00:00Z',
          mid: '100',
          bands: [
            { rangePct: '0.1', bid: '20', ask: '10' },
            { rangePct: '1', bid: '80', ask: '40' },
          ],
        },
      ],
      gaps: [{ time: '2024-01-01T00:01:00Z', missing: ['okx'], stale: [] }],
    };
    mockPerpBook.mockReturnValue({
      data: book,
      currentData: book,
      isError: false,
      isFetching: false,
      isLoading: false,
      refetch: vi.fn(),
    });
    mockPerpDepth.mockReturnValue({
      data: depth,
      currentData: depth,
      isError: false,
      isFetching: false,
      isLoading: false,
      refetch: vi.fn(),
    });
    mockOrderBook.mockReturnValue({
      data: {
        analysis: {
          walls: [{ side: 'bid', price: '100', notional: '250000', behavior: 'persistent' }],
        },
      },
      currentData: {
        analysis: {
          walls: [{ side: 'bid', price: '100', notional: '250000', behavior: 'persistent' }],
        },
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });

    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByRole('radio', { name: 'All' })).toBeChecked();

    await chooseDesk(user, 'Positioning');
    expect(screen.getByRole('switch', { name: /open interest/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /funding/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /crowd vs top/i })).toBeChecked();
    expect(screen.getByTestId('chart-no-top')).toHaveTextContent(/Bybit/);
    expect(screen.getByTestId('chart-no-top')).toHaveTextContent(/top-trader ratio/);
    expect(screen.getByRole('switch', { name: /open levels/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /liquidation levels/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /liq map/i })).not.toBeChecked();
    expect(screen.getByRole('switch', { name: /^book$/i })).not.toBeChecked();
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-liq', '0');

    await chooseDesk(user, 'Order flow');
    expect(screen.getByRole('switch', { name: /buy − sell/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /^cvd$/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /footprint/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /open interest/i })).not.toBeChecked();
    expect(screen.getByTestId('chart-preset-absorption')).toHaveTextContent('bids absorbing sells');
    expect(screen.getByTestId('chart-preset-absorption')).toHaveTextContent('OKX');
    expect(screen.getByTestId('chart-preset-print-size')).toHaveTextContent('Last 15m: buy prints averaged 2K USDT.');
    expect(screen.getByTestId('chart-preset-print-size')).toHaveTextContent('Bybit');

    await chooseDesk(user, 'Liquidity');
    expect(screen.getByRole('switch', { name: /^book$/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /liq map/i })).toBeChecked();
    expect(screen.getByRole('switch', { name: /footprint/i })).not.toBeChecked();
    expect(screen.getByTestId('chart-preset-bid')).toHaveTextContent(/Bybit/);
    expect(screen.getByTestId('chart-preset-bid')).toHaveTextContent(/OKX/);
    expect(screen.queryByText(/Perp bids are 60%/)).not.toBeInTheDocument();
    expect(screen.getByTestId('chart-preset-bid-recorded')).toHaveTextContent(/Binance 60%/);
    expect(screen.getByTestId('chart-preset-walls')).toHaveTextContent(/Bid 100/);
    expect(screen.getByTestId('chart-preset-walls')).toHaveTextContent(/Resting/);
    expect(screen.getByTestId('chart-perp-book')).toHaveTextContent(/OKX/);
    expect(Number(screen.getByTestId('candle-chart').getAttribute('data-liq'))).toBeGreaterThan(0);

    await user.click(screen.getByRole('switch', { name: /footprint/i }));
    expect(screen.getByRole('radio', { name: 'Liquidity' })).not.toBeChecked();
    expect(screen.getByRole('radio', { name: 'All' })).not.toBeChecked();
  });

  it('draws the stored perp book behind the liquidation map', async () => {
    const book = {
      stepSec: 60,
      columns: [
        {
          time: '2026-10-09T12:00:00Z',
          mid: '100',
          coverBid: '0.15',
          coverAsk: '0.2',
          bands: [
            { rangePct: '0.1', bid: '10', ask: '5' },
            { rangePct: '0.5', bid: '30', ask: '15' },
            { rangePct: '1', bid: '40', ask: '20' },
          ],
        },
      ],
      gaps: [],
    };
    mockPerpBook.mockReturnValue({
      data: book,
      currentData: book,
      isError: false,
      isFetching: false,
      isLoading: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByRole('radio', { name: 'All' })).toBeChecked();
    expect(screen.getByRole('switch', { name: /liq map/i })).toBeChecked();
    expect(screen.getByTestId('chart-perp-book')).toBeInTheDocument();
    expect(screen.getByTestId('chart-perp-book-reach')).toHaveTextContent('0.15%');
    expect(screen.getByTestId('chart-perp-book-reach')).toHaveTextContent('0.2%');
    expect(Number(screen.getByTestId('candle-chart').getAttribute('data-liq'))).toBeGreaterThan(4);
  });

  it('paints stored buy and sell inside candles and names a venue that did not record', async () => {
    const report = {
      symbol: 'BTCUSDT',
      exchange: 'all',
      interval: '1h',
      stepSec: 3600,
      definite: false,
      truncated: true,
      missing: ['okx'],
      venues: [
        {
          exchange: 'binance',
          candles: [
            {
              time: '2024-01-01T00:00:00Z',
              rows: [
                {
                  price: '100',
                  high: '101',
                  buyVolume: '3',
                  sellVolume: '1',
                  isPoc: true,
                  imbalance: 'buy',
                },
              ],
            },
          ],
        },
      ],
    };
    mockCandleFootprint.mockReturnValue({
      data: report,
      currentData: report,
      isError: false,
      isFetching: false,
      isLoading: false,
      isSuccess: true,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByTestId('chart-candle-footprint')).toBeInTheDocument();
    expect(screen.getByTestId('chart-candle-footprint-venue')).toHaveTextContent(/OKX/);
    expect(screen.getByTestId('chart-candle-footprint-venue')).toHaveTextContent(/Binance/);
    expect(screen.getByTestId('chart-candle-footprint-truncated')).toBeInTheDocument();
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-candle-foot', '1');
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-foot-strip', '0');
    expect(mockCandleFootprint).toHaveBeenCalledWith(
      expect.objectContaining({ exchange: 'all', symbol: 'BTCUSDT', interval: '1h' }),
      expect.anything(),
    );
  });

  it('restarts CVD at a clicked candle and clears it on the same candle', async () => {
    const user = userEvent.setup();
    const open = new Date(Date.now() - 2 * 60 * 60 * 1000);
    open.setUTCMinutes(0, 0, 0);
    chartClickAt = Math.floor(open.getTime() / 1000);
    const openIso = open.toISOString();
    mockCandles.mockReturnValue({
      data: { candles: [{ ...candle, openTime: openIso }] },
      currentData: { candles: [{ ...candle, openTime: openIso }] },
      isLoading: false,
      isError: false,
      isSuccess: true,
      isFetching: false,
      refetch: vi.fn(),
    });
    const cvd = {
      range: '24h',
      bar: '5m',
      combined: { complete: false, points: [] },
      venues: [
        {
          exchange: 'binance',
          complete: true,
          points: [
            { time: openIso, delta: '+5' },
            { time: new Date(open.getTime() + 5 * 60 * 1000).toISOString(), delta: '-2' },
          ],
        },
        { exchange: 'bybit', points: [] },
        { exchange: 'okx', points: [] },
      ],
    };
    mockCvd.mockReturnValue({
      data: cvd,
      currentData: cvd,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByTestId('chart-cvd-click-hint')).toBeInTheDocument();
    await user.click(screen.getByTestId('chart-bar-click'));
    const caption = await screen.findByTestId('chart-cvd-click');
    const pad = (n: number) => String(n).padStart(2, '0');
    const stamp = `${open.getUTCFullYear()}-${pad(open.getUTCMonth() + 1)}-${pad(open.getUTCDate())} ${pad(open.getUTCHours())}:${pad(open.getUTCMinutes())}`;
    expect(caption).toHaveTextContent(stamp);
    expect(caption).toHaveTextContent('Binance');
    expect(caption).toHaveTextContent('+3');
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-verts', expect.stringContaining('CVD'));
    await user.click(screen.getByTestId('chart-bar-click'));
    expect(screen.queryByTestId('chart-cvd-click')).not.toBeInTheDocument();
    expect(screen.getByTestId('chart-cvd-click-hint')).toBeInTheDocument();
  });

  it('shows the forensics read for a clicked candle', async () => {
    const user = userEvent.setup();
    const open = new Date(Date.now() - 2 * 60 * 60 * 1000);
    open.setUTCMinutes(0, 0, 0);
    chartClickAt = Math.floor(open.getTime() / 1000);
    const openIso = open.toISOString();
    mockCandles.mockReturnValue({
      data: { candles: [{ ...candle, openTime: openIso }] },
      currentData: { candles: [{ ...candle, openTime: openIso }] },
      isLoading: false,
      isError: false,
      isSuccess: true,
      isFetching: false,
      refetch: vi.fn(),
    });
    mockForensics.mockReturnValue({
      data: undefined,
      currentData: {
        why: {
          headline: 'Price rose while open interest fell.',
          before: 'Bids had thinned before the candle.',
        },
        timeline: {
          events: [{ at: openIso, title: 'Binance open interest started down', offsetLabel: '-8m' }],
        },
        similar: {
          summary: 'One similar setup sold off afterward.',
          matches: [
            {
              at: '2024-06-01T12:00:00Z',
              until: '2024-06-01T12:15:00Z',
              direction: 'down',
              similarity: '70',
              coverage: '80',
              afterReturnPct: '-1.2',
            },
          ],
        },
      },
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    expect(await screen.findByTestId('chart-forensics-hint')).toBeInTheDocument();
    await user.click(screen.getByTestId('chart-bar-click'));
    expect(await screen.findByTestId('chart-forensics')).toHaveTextContent(
      'Price rose while open interest fell.',
    );
    expect(screen.getByTestId('chart-forensics')).toHaveTextContent('One similar setup sold off afterward.');
    expect(mockForensics).toHaveBeenCalledWith(
      expect.objectContaining({
        symbol: 'BTCUSDT',
        at: openIso,
        exchange: 'all',
        window: '1h',
        during: '15m',
      }),
      expect.objectContaining({ skip: false }),
    );
    await user.click(screen.getByText('-1.2'));
    expect(screen.getByTestId('chart-forensics')).toHaveTextContent('2024-06-01 12:00');
    await user.click(screen.getByTestId('chart-forensics-clear'));
    expect(screen.queryByTestId('chart-forensics')).not.toBeInTheDocument();
    expect(screen.getByTestId('chart-forensics-hint')).toBeInTheDocument();
  });

  it('restarts open interest, funding, and liquidations from a clicked candle', async () => {
    const user = userEvent.setup();
    const open = new Date(Date.now() - 2 * 60 * 60 * 1000);
    open.setUTCMinutes(0, 0, 0);
    chartClickAt = Math.floor(open.getTime() / 1000);
    const openIso = open.toISOString();
    mockCandles.mockReturnValue({
      data: { candles: [{ ...candle, openTime: openIso }] },
      currentData: { candles: [{ ...candle, openTime: openIso }] },
      isLoading: false,
      isError: false,
      isSuccess: true,
      isFetching: false,
      refetch: vi.fn(),
    });
    const report = {
      anchor: openIso,
      openInterest: [
        {
          exchange: 'binance',
          complete: true,
          change: '6',
          points: [
            { time: openIso, value: '0' },
            { time: new Date(open.getTime() + 60 * 60 * 1000).toISOString(), value: '6' },
          ],
        },
      ],
      funding: [],
      liquidations: [{ exchange: 'binance', complete: true, change: '0', points: [{ time: openIso, value: '0' }] }],
      missingOpenInterest: ['bybit', 'okx'],
      missingFunding: ['binance', 'bybit', 'okx'],
      missingLiquidations: ['bybit', 'okx'],
      lateLiquidations: [],
    };
    mockPositioningAnchor.mockReturnValue({
      data: report,
      currentData: report,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    await chooseDesk(user, 'Positioning');
    expect(await screen.findByTestId('chart-positioning-click-hint')).toBeInTheDocument();
    await user.click(screen.getByTestId('chart-bar-click'));
    const caption = await screen.findByTestId('chart-positioning-click');
    expect(caption).toHaveTextContent('Binance 6');
    expect(caption).toHaveTextContent('Bybit');
    expect(caption).toHaveTextContent('OKX');
    expect(caption).toHaveTextContent(/stays off/i);
    expect(screen.getByTestId('candle-chart')).toHaveAttribute('data-verts', expect.stringContaining('Anchor'));
    await user.click(screen.getByTestId('chart-positioning-click-clear'));
    expect(screen.queryByTestId('chart-positioning-click')).not.toBeInTheDocument();
    expect(screen.getByTestId('chart-positioning-click-hint')).toBeInTheDocument();
  });

  it('marks the current price and CVD disagreement and hides it on positioning', async () => {
    const user = userEvent.setup();
    const cvd = {
      range: '24h',
      combined: {
        complete: false,
        points: [{ time: '2024-01-01T00:00:00Z', delta: '+1' }],
      },
      venues: [
        {
          exchange: 'binance',
          complete: true,
          points: [{ time: '2024-01-01T00:00:00Z', delta: '-4' }],
          divergence: {
            kind: 'price_up_cvd_down',
            since: '2024-01-01T00:00:00Z',
            duration: '35m',
            durationSeconds: 2100,
          },
        },
        { exchange: 'bybit', points: [] },
        { exchange: 'okx', points: [] },
      ],
    };
    mockCvd.mockReturnValue({
      data: cvd,
      currentData: cvd,
      isLoading: false,
      isError: false,
      isFetching: false,
      refetch: vi.fn(),
    });
    renderDetail('/markets/binance/BTCUSDT');
    const caption = await screen.findByTestId('chart-cvd-split');
    expect(caption).toHaveTextContent('Price up, CVD down for 35m on Binance.');
    expect(caption).toHaveTextContent('Bybit');
    expect(caption).toHaveTextContent('OKX');
    expect(screen.getByTestId('candle-chart')).toHaveAttribute(
      'data-marks',
      expect.stringContaining('Price up, CVD down'),
    );

    await chooseDesk(user, 'Positioning');
    expect(screen.queryByTestId('chart-cvd-split')).not.toBeInTheDocument();
    expect(screen.getByTestId('candle-chart').getAttribute('data-marks') ?? '').not.toContain(
      'Price up, CVD down',
    );

    await chooseDesk(user, 'Order flow');
    expect(screen.getByTestId('chart-cvd-split')).toHaveTextContent('Price up, CVD down for 35m on Binance.');
  });

  it('opens holders tab from the URL', async () => {
    renderDetail('/markets/binance/BTCUSDT?tab=holders');
    expect(await screen.findByText(/On-chain wallet count and top addresses/i)).toBeInTheDocument();
    expect(screen.queryByTestId('candle-chart')).not.toBeInTheDocument();
  });

  it('opens order book tab from the URL', async () => {
    renderDetail('/markets/binance/BTCUSDT?tab=orderbook');
    expect(await screen.findByTestId('order-book')).toBeInTheDocument();
    expect(screen.getByTestId('order-depth-chart')).toBeInTheDocument();
    expect(screen.getByTestId('order-heatmap')).toBeInTheDocument();
    expect(screen.getByTestId('book-gaps')).toBeInTheDocument();
  });

  it('opens tape tab with liquidation totals', async () => {
    renderDetail('/markets/binance/BTCUSDT?tab=tape');
    expect(await screen.findAllByText(/^Liquidations$|^Likidasyonlar$/i)).not.toHaveLength(0);
    expect(screen.queryByTestId('liquidation-heatmap')).not.toBeInTheDocument();
  });

  it('opens flow and structure tabs from the URL', async () => {
    renderDetail('/markets/binance/BTCUSDT?tab=flow');
    expect(await screen.findByTestId('coin-flow-tab')).toBeInTheDocument();
    expect(screen.queryByTestId('coin-structure-tab')).not.toBeInTheDocument();
    renderDetail('/markets/binance/BTCUSDT?tab=structure');
    expect(await screen.findByTestId('coin-structure-tab')).toBeInTheDocument();
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
    renderDetail('/markets/binance/BTCUSDT?tab=indicators');
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
