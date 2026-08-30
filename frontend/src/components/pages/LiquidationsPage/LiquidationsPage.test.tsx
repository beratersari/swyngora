import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { LiquidationsPage } from './LiquidationsPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketLiquidationOverviewQuery: () => ({
      data: {
        exchange: 'all',
        coinWindow: '24h',
        windows: [
          { window: '1h', totalNotional: '100', longNotional: '60', shortNotional: '40', count: 2, complete: true },
          { window: '4h', totalNotional: '200', longNotional: '110', shortNotional: '90', count: 4, complete: true },
          { window: '12h', totalNotional: '300', longNotional: '140', shortNotional: '160', count: 6, complete: true },
          { window: '24h', totalNotional: '400', longNotional: '180', shortNotional: '220', count: 8, complete: true },
        ],
        coins: [
          { symbol: 'BTCUSDT', base: 'BTC', totalNotional: '300', longNotional: '200', shortNotional: '100' },
        ],
      },
      currentData: {
        exchange: 'all',
        coinWindow: '24h',
        windows: [
          { window: '1h', totalNotional: '100', longNotional: '60', shortNotional: '40', count: 2, complete: true },
          { window: '4h', totalNotional: '200', longNotional: '110', shortNotional: '90', count: 4, complete: true },
          { window: '12h', totalNotional: '300', longNotional: '140', shortNotional: '160', count: 6, complete: true },
          { window: '24h', totalNotional: '400', longNotional: '180', shortNotional: '220', count: 8, complete: true },
        ],
        coins: [
          { symbol: 'BTCUSDT', base: 'BTC', totalNotional: '300', longNotional: '200', shortNotional: '100' },
        ],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetMarketLiquidationHuntHeatmapQuery: () => ({
      data: undefined,
      currentData: undefined,
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetTicker24hQuery: () => ({
      data: { lastPrice: '64000' },
      currentData: { lastPrice: '64000' },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
  };
});

describe('LiquidationsPage', () => {
  it('renders overview cards and the coin map', async () => {
    renderWithProviders(<LiquidationsPage />, { routerEntries: ['/liquidations'] });
    expect(await screen.findByRole('heading', { name: /liquidations|likidasyonlar/i })).toBeInTheDocument();
    expect(screen.getByRole('group', { name: /liquidation coin map|likidasyon coin/i })).toBeInTheDocument();
    expect(screen.getAllByText(/total|toplam/i).length).toBeGreaterThan(0);
  });

  it('opens the heatmap tab from the URL', async () => {
    renderWithProviders(<LiquidationsPage />, { routerEntries: ['/liquidations?view=heatmap'] });
    expect(await screen.findByTestId('liquidation-heatmap')).toBeInTheDocument();
  });
});
