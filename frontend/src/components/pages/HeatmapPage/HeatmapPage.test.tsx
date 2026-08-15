import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { HeatmapPage } from './HeatmapPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListSpotMarketsQuery: () => ({
      data: {
        items: [
          {
            symbol: 'BTCUSDT',
            lastPrice: '100',
            priceChangePercent: '1.5',
            quoteVolume: '5000',
          },
        ],
      },
      currentData: {
        items: [
          {
            symbol: 'BTCUSDT',
            lastPrice: '100',
            priceChangePercent: '1.5',
            quoteVolume: '5000',
          },
        ],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
  };
});

vi.mock('@/libs/realtime', () => ({
  usePriceSubscription: () => ({ connected: false }),
  useRealtimeConnected: () => false,
}));

describe('HeatmapPage', () => {
  it('renders title and heatmap tiles', async () => {
    renderWithProviders(<HeatmapPage />, { routerEntries: ['/heatmap'] });
    expect(await screen.findByRole('heading', { name: /heatmap|ısı haritası/i })).toBeInTheDocument();
    expect(screen.getByRole('group', { name: /heatmap|ısı/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /fullscreen|tam ekran/i })).toBeInTheDocument();
  });
});
