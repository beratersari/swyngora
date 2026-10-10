import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { TapeBacktestPanel } from './TapeBacktestPanel';

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    useGetMarketTapeBacktestQuery: () => ({
      isFetching: false,
      isError: false,
      data: {
        definite: true,
        condition: 'funding',
        summary: '1 longs-pay funding crosses.',
        venues: [
          {
            exchange: 'binance',
            signalCount: 1,
            bars: 2,
            horizons: [
              { horizon: '1h', sampleCount: 1, average: 1.2, median: 1.2, winRate: 100, worstDrop: -2.5 },
              { horizon: '4h', sampleCount: 0 },
              { horizon: '24h', sampleCount: 0 },
            ],
            signals: [],
          },
        ],
      },
    }),
  };
});

describe('TapeBacktestPanel', () => {
  it('shows the worst drop for each horizon', () => {
    renderWithProviders(<TapeBacktestPanel />);
    expect(screen.getByText('Worst drop')).toBeInTheDocument();
    expect(screen.getByText('-2.50%')).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Condition' })).toBeInTheDocument();
  });
});
