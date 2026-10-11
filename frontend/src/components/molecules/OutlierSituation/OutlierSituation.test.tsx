import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OutlierSituation } from './OutlierSituation';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketSituationQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        priceCvd: { kind: 'agrees' },
        prints: { side: 'short', longNotional: 3200, shortNotional: 2_500_000, window: '24h' },
      },
      isLoading: false,
      isFetching: false,
      isError: false,
    }),
  };
});

describe('OutlierSituation', () => {
  it('shows the combined situation clauses for the opened coin', () => {
    renderWithProviders(<OutlierSituation symbol="BTCUSDT" />);
    expect(screen.getByTestId('outlier-situation')).toBeInTheDocument();
    expect(screen.getByText('Combined situation for this coin.')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
    expect(
      screen.getByText('Measured prints over the last 24h are mostly short liquidations (3.20K long, 2.50M short).'),
    ).toBeInTheDocument();
  });
});
