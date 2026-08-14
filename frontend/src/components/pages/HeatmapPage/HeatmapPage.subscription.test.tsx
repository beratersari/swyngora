import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { HeatmapPage } from './HeatmapPage';

const usePriceSubscription = vi.hoisted(() => vi.fn(() => ({ connected: true })));

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
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
  };
});

vi.mock('@/libs/realtime', () => ({
  usePriceSubscription: (...args: unknown[]) => usePriceSubscription(...args),
  useRealtimeConnected: () => true,
}));

describe('HeatmapPage subscription', () => {
  it('subscribes to {exchange, symbol} refs, not the venue string', async () => {
    renderWithProviders(<HeatmapPage />, { routerEntries: ['/heatmap'] });
    expect(usePriceSubscription).toHaveBeenCalled();
    const first = usePriceSubscription.mock.calls[0]?.[0];
    expect(Array.isArray(first)).toBe(true);
    expect(first).toEqual([{ exchange: 'binance', symbol: 'BTCUSDT' }]);
  });
});
