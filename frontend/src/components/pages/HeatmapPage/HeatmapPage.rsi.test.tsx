import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { HeatmapPage } from './HeatmapPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useListSpotMarketsQuery: () => ({
      data: { items: [] },
      currentData: { items: [] },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetRSIHeatmapQuery: () => ({
      data: {
        exchange: 'binance',
        interval: '1h',
        averageRsi: 31,
        items: [{ rank: 1, symbol: 'ETHUSDT', base: 'ETH', rsi: 31, zone: 'neutral' }],
      },
      currentData: {
        exchange: 'binance',
        interval: '1h',
        averageRsi: 31,
        items: [{ rank: 1, symbol: 'ETHUSDT', base: 'ETH', rsi: 31, zone: 'neutral' }],
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

describe('HeatmapPage RSI view', () => {
  it('shows the RSI scatter when view=rsi', async () => {
    renderWithProviders(<HeatmapPage />, { routerEntries: ['/heatmap?view=rsi'] });
    expect(await screen.findByTestId('rsi-heatmap')).toBeInTheDocument();
    expect(screen.getByText('ETH')).toBeInTheDocument();
    expect(screen.getByTestId('rsi-avg-line')).toBeInTheDocument();
  });
});
