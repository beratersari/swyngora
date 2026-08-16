import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { WatchlistDeskTape } from './WatchlistDeskTape';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetWatchlistQuery: () => ({
      data: {
        items: [
          { exchange: 'binance', symbol: 'RAREUSDT' },
          { exchange: 'bist', symbol: 'THYAO' },
        ],
      },
      isLoading: false,
      isFetching: false,
    }),
    useGetTicker24hQuery: ({ symbol }: { symbol: string }) => ({
      data:
        symbol === 'RAREUSDT'
          ? { lastPrice: '0.12', priceChangePercent: '3.1' }
          : { lastPrice: '285.5', priceChangePercent: '-0.4' },
      isLoading: false,
    }),
  };
});

describe('WatchlistDeskTape', () => {
  it('quotes every watched symbol, not only top-volume names', () => {
    renderWithProviders(<WatchlistDeskTape ariaLabel="Live market prices" emptyLabel="empty" />);
    expect(screen.getAllByRole('link', { name: /RARE\/USDT 0\.12/i }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('link', { name: /THYAO 285\.5/i }).length).toBeGreaterThan(0);
  });
});
