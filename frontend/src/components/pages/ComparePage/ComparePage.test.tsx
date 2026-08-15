import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ComparePage } from './ComparePage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetCandlesQuery: () => ({
      data: {
        candles: [
          {
            openTime: '2024-01-01T00:00:00Z',
            open: '100',
            high: '110',
            low: '90',
            close: '100',
            volume: '1',
          },
          {
            openTime: '2024-01-01T01:00:00Z',
            open: '100',
            high: '120',
            low: '100',
            close: '110',
            volume: '1',
          },
        ],
      },
      currentData: {
        candles: [
          {
            openTime: '2024-01-01T00:00:00Z',
            open: '100',
            high: '110',
            low: '90',
            close: '100',
            volume: '1',
          },
          {
            openTime: '2024-01-01T01:00:00Z',
            open: '100',
            high: '120',
            low: '100',
            close: '110',
            volume: '1',
          },
        ],
      },
      isLoading: false,
      isError: false,
    }),
  };
});

vi.mock('@/components/molecules/CompareChartHost', () => ({
  CompareChartHost: () => <div data-testid="compare-chart" />,
  SERIES_COLORS: ['#0f0', '#0ff', '#ff0'],
}));

describe('ComparePage', () => {
  it('renders title and add controls', async () => {
    renderWithProviders(<ComparePage />, {
      routerEntries: ['/compare?pairs=binance:BTCUSDT'],
    });
    expect(await screen.findByText(/compare|karşılaştır/i)).toBeInTheDocument();
    expect(screen.getByTestId('compare-chart')).toBeInTheDocument();
  });
});
