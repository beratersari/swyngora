import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { MarketsPage } from './MarketsPage';

const mockSpot = vi.fn();
const mockExchanges = vi.fn();
const mockTags = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListExchangesQuery: () => mockExchanges(),
    useListProductTagsQuery: () => mockTags(),
    useListSpotMarketsQuery: () => mockSpot(),
    useListDelistScheduleQuery: () => ({
      data: { exchange: 'binance', enabled: true, items: [] },
      isSuccess: true,
      isLoading: false,
    }),
    useGetWatchlistQuery: () => ({ data: { items: [] }, isLoading: false }),
    useAddWatchlistItemMutation: () => [vi.fn(), { isLoading: false }],
    useRemoveWatchlistItemMutation: () => [vi.fn(), { isLoading: false }],
  };
});

const binanceItem = {
  symbol: 'BTCUSDT',
  baseAsset: 'BTC',
  quoteAsset: 'USDT',
  lastPrice: '67000',
  priceChangePercent: '1.2',
  quoteVolume: '1000000',
};

const coinbaseItem = {
  symbol: 'BTC-USD',
  baseAsset: 'BTC',
  quoteAsset: 'USD',
  lastPrice: '67000',
  priceChangePercent: '0.5',
  quoteVolume: '900000',
};

function okSpot(items: typeof binanceItem[], total = items.length) {
  const payload = { items, total };
  return {
    data: payload,
    currentData: payload,
    isLoading: false,
    isFetching: false,
    isSuccess: true,
    isError: false,
    error: undefined,
    refetch: vi.fn(),
  };
}

describe('MarketsPage', () => {
  beforeEach(() => {
    mockExchanges.mockReturnValue({
      data: { exchanges: ['binance', 'coinbase'], default: 'binance' },
      isLoading: false,
      isError: false,
    });
    mockTags.mockReturnValue({ data: { tags: [], exchange: 'binance' }, isFetching: false });
    mockSpot.mockReturnValue(okSpot([binanceItem]));
  });

  it('renders table rows from spot query', async () => {
    renderWithProviders(<MarketsPage />, { routerEntries: ['/markets'] });
    await waitFor(() => {
      expect(screen.getByText('BTC/USDT')).toBeInTheDocument();
    });
  });

  it('shows error state when spot fails with no rows', async () => {
    mockSpot.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      isSuccess: false,
      isError: true,
      error: { status: 500, data: {} },
      refetch: vi.fn(),
    });
    renderWithProviders(<MarketsPage />, { routerEntries: ['/markets'] });
    expect(await screen.findByText('Could not load markets')).toBeInTheDocument();
  });

  it('keeps rows and shows refresh warning when poll fails with prior data', async () => {
    mockSpot.mockReturnValue({
      data: { items: [binanceItem], total: 1 },
      currentData: { items: [binanceItem], total: 1 },
      isLoading: false,
      isFetching: false,
      isSuccess: false,
      isError: true,
      error: { status: 502, data: {} },
      refetch: vi.fn(),
    });
    renderWithProviders(<MarketsPage />, { routerEntries: ['/markets'] });
    expect(await screen.findByText('BTC/USDT')).toBeInTheDocument();
    expect(
      await screen.findByText(/Could not refresh markets/i),
    ).toBeInTheDocument();
  });

  it('shows mcap warmup alert when marketCap sort errors', async () => {
    mockSpot.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      isSuccess: false,
      isError: true,
      error: { status: 502, data: {} },
      refetch: vi.fn(),
    });
    renderWithProviders(<MarketsPage />, {
      routerEntries: ['/markets?sort=marketCapCirculating'],
    });
    expect(
      await screen.findByText(/Market-cap data may be warming up/i),
    ).toBeInTheDocument();
  });

  it('switches exchange and shows new venue rows when data arrives', async () => {
    const user = userEvent.setup();
    mockSpot.mockReturnValue(okSpot([binanceItem]));
    renderWithProviders(<MarketsPage />, { routerEntries: ['/markets'] });
    expect(await screen.findByText('BTC/USDT')).toBeInTheDocument();

    mockSpot.mockReturnValue(okSpot([coinbaseItem]));
    await user.click(screen.getByRole('tab', { name: /coinbase/i }));
    // Coinbase BTC-USD displays as BTC/USD (same BASE/QUOTE convention)
    expect(await screen.findByText('BTC/USD')).toBeInTheDocument();
  });

  it('resets quote to USD when switching to Coinbase', async () => {
    const user = userEvent.setup();
    mockSpot.mockReturnValue(okSpot([binanceItem]));
    renderWithProviders(<MarketsPage />, {
      routerEntries: ['/markets?quote=USDT'],
    });
    expect(await screen.findByText('BTC/USDT')).toBeInTheDocument();

    mockSpot.mockReturnValue(okSpot([coinbaseItem]));
    await user.click(screen.getByRole('tab', { name: /coinbase/i }));
    // Meta line shows venue default quote after switch
    await waitFor(() => {
      expect(screen.getByText(/Quote USD/i)).toBeInTheDocument();
    });
  });

  it('does not paint previous venue rows when filter changes before new data', async () => {
    const user = userEvent.setup();
    mockSpot.mockReturnValue(okSpot([binanceItem]));
    renderWithProviders(<MarketsPage />, { routerEntries: ['/markets'] });
    expect(await screen.findByText('BTC/USDT')).toBeInTheDocument();

    // After tab click, URL exchange changes; mock returns no data for new key
    mockSpot.mockImplementation(() => ({
      data: { items: [binanceItem], total: 1 },
      currentData: undefined,
      isLoading: false,
      isFetching: true,
      isSuccess: false,
      isError: false,
      error: undefined,
      refetch: vi.fn(),
    }));
    await user.click(screen.getByRole('tab', { name: /coinbase/i }));
    await waitFor(() => {
      expect(screen.queryByText('BTC/USDT')).not.toBeInTheDocument();
    });
  });

  it('shows updating tag while refreshing with rows', async () => {
    mockSpot.mockReturnValue({
      ...okSpot([binanceItem]),
      isFetching: true,
    });
    renderWithProviders(<MarketsPage />, { routerEntries: ['/markets'] });
    expect(await screen.findByText(/updating/i)).toBeInTheDocument();
  });

  it('falls back to default exchanges when list empty', async () => {
    mockExchanges.mockReturnValue({
      data: { exchanges: [], default: 'binance' },
      isLoading: false,
      isError: true,
    });
    renderWithProviders(<MarketsPage />, { routerEntries: ['/markets'] });
    expect(await screen.findByRole('tab', { name: /binance/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /coinbase/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /bybit/i })).toBeInTheDocument();
  });
});
