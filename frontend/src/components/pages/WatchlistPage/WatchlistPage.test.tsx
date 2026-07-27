import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { WatchlistPage } from './WatchlistPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetWatchlistQuery: () => ({
      data: {
        clientId: 't',
        items: [{ exchange: 'binance', symbol: 'BTCUSDT', addedAt: '2024-01-01T00:00:00Z' }],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useRemoveWatchlistItemMutation: () => [vi.fn(), { isLoading: false }],
    useListSpotMarketsQuery: () => ({
      data: {
        items: [
          {
            symbol: 'BTCUSDT',
            lastPrice: '65000',
            priceChangePercent: '1.5',
            quoteVolume: '1000000000',
            marketCapCirculating: 1.2e12,
          },
        ],
      },
      isLoading: false,
      isFetching: false,
    }),
  };
});

describe('WatchlistPage', () => {
  it('shows pair and live market columns', async () => {
    renderWithProviders(<WatchlistPage />, { routerEntries: ['/watchlist'] });
    expect(await screen.findByText('BTC/USDT')).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /last/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /24h/i })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /mcap|circ/i })).toBeInTheDocument();
    // price / change / mcap from mock (locale-aware)
    expect(screen.getByText(/65/)).toBeInTheDocument();
    expect(screen.getByText(/\+1\.50%/)).toBeInTheDocument();
    expect(screen.getByText(/1\.20T|1\.2T/)).toBeInTheDocument();
  });
});
