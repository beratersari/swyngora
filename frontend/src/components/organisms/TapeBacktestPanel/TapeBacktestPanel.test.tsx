import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { TapeBacktestPanel } from './TapeBacktestPanel';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useLazyListSpotMarketsQuery: () => [vi.fn(), { data: undefined, isFetching: false }],
    useGetMarketTapeBacktestQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        condition: 'cvd',
        side: 'buy',
        definite: false,
        missing: ['okx'],
        summary: '1 buy-share crosses. combined withheld (okx).',
        note: 'Informational only — not financial advice.',
        venues: [
          {
            exchange: 'binance',
            signalCount: 1,
            bars: 2,
            horizons: [
              { horizon: '1h', sampleCount: 1, average: 1.5, median: 1.5, winRate: 100 },
              { horizon: '4h', sampleCount: 0 },
              { horizon: '24h', sampleCount: 0 },
            ],
            signals: [{ at: '2026-10-05T12:00:00.000Z', value: 0.7, close: 100, return1h: 1.5 }],
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

describe('TapeBacktestPanel', () => {
  it('shows the withheld combined read and the next-hour return', async () => {
    renderWithProviders(<TapeBacktestPanel />);
    expect(await screen.findByRole('heading', { name: /stored tape|kayıtlı bant/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /replay|yeniden oynat/i })).toBeInTheDocument();
    expect(screen.getAllByText(/okx/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText('+1.50%').length).toBeGreaterThan(0);
    expect(screen.getByText('70.0%')).toBeInTheDocument();
  });
});
