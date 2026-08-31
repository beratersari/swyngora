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
    useGetMarketLiquidationLevelsQuery: () => ({
      data: {
        kind: 'levels',
        symbol: 'BTCUSDT',
        lastPrice: '100',
        levels: [{ price: '100', longNotional: '10', shortNotional: '5', totalNotional: '15' }],
      },
      currentData: {
        kind: 'levels',
        symbol: 'BTCUSDT',
        lastPrice: '100',
        levels: [{ price: '100', longNotional: '10', shortNotional: '5', totalNotional: '15' }],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetMarketLiquidationCascadeQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        summary: 'No liquidation cascade on this coin.',
        venues: [
          { exchange: 'binance', grade: 'quiet', side: 'none', windows: [] },
          { exchange: 'bybit', grade: 'quiet', side: 'none', windows: [] },
        ],
      },
      currentData: {
        symbol: 'BTCUSDT',
        summary: 'No liquidation cascade on this coin.',
        venues: [
          { exchange: 'binance', grade: 'quiet', side: 'none', windows: [] },
          { exchange: 'bybit', grade: 'quiet', side: 'none', windows: [] },
        ],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetMarketLiquidationCascadeScanQuery: () => ({
      data: {
        market: {
          symbol: 'all',
          summary: 'Market liquidation flow is in a normal range.',
          venues: [
            { exchange: 'binance', grade: 'quiet', side: 'none', windows: [] },
            { exchange: 'bybit', grade: 'quiet', side: 'none', windows: [] },
          ],
        },
        hits: [],
      },
      currentData: {
        market: {
          symbol: 'all',
          summary: 'Market liquidation flow is in a normal range.',
          venues: [
            { exchange: 'binance', grade: 'quiet', side: 'none', windows: [] },
            { exchange: 'bybit', grade: 'quiet', side: 'none', windows: [] },
          ],
        },
        hits: [],
      },
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
    useGetMarketLiquidationHuntQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        bias: { lean: 'up', summary: 'Up looks easier (68 vs 41).' },
        venues: [
          {
            exchange: 'binance',
            price: '64000',
            upScore: { direction: 'up', score: 68, level: 'likely', reasons: ['Shorts crowded'] },
            downScore: { direction: 'down', score: 41, level: 'mixed', reasons: [] },
            upHunt: { target: { price: '65000', movePct: '1.5' }, houseEdge: 'profit' },
            downHunt: { target: { price: '63000', movePct: '-1.5' }, houseEdge: 'loss' },
            upCascade: {
              direction: 'up',
              summary: '2 short liquidations above zones.',
              steps: [{ index: 1, band: { price: '64256', leverage: '125' }, zoneNotional: '800000', reachable: true }],
            },
            downCascade: { direction: 'down', steps: [] },
          },
        ],
      },
      currentData: {
        symbol: 'BTCUSDT',
        bias: { lean: 'up', summary: 'Up looks easier (68 vs 41).' },
        venues: [
          {
            exchange: 'binance',
            price: '64000',
            upScore: { direction: 'up', score: 68, level: 'likely', reasons: ['Shorts crowded'] },
            downScore: { direction: 'down', score: 41, level: 'mixed', reasons: [] },
            upHunt: { target: { price: '65000', movePct: '1.5' }, houseEdge: 'profit' },
            downHunt: { target: { price: '63000', movePct: '-1.5' }, houseEdge: 'loss' },
            upCascade: {
              direction: 'up',
              summary: '2 short liquidations above zones.',
              steps: [{ index: 1, band: { price: '64256', leverage: '125' }, zoneNotional: '800000', reachable: true }],
            },
            downCascade: { direction: 'down', steps: [] },
          },
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

  it('opens the chart tab from the URL', async () => {
    renderWithProviders(<LiquidationsPage />, { routerEntries: ['/liquidations?view=chart'] });
    expect(await screen.findByTestId('liquidation-bar-chart')).toBeInTheDocument();
  });

  it('opens the cascade tab from the URL', async () => {
    renderWithProviders(<LiquidationsPage />, { routerEntries: ['/liquidations?view=cascade'] });
    expect(await screen.findByTestId('liquidation-cascade')).toBeInTheDocument();
  });

  it('opens the heatmap tab from the URL', async () => {
    renderWithProviders(<LiquidationsPage />, { routerEntries: ['/liquidations?view=heatmap'] });
    expect(await screen.findByTestId('liquidation-heatmap')).toBeInTheDocument();
  });

  it('opens the hunt tab from the URL', async () => {
    renderWithProviders(<LiquidationsPage />, { routerEntries: ['/liquidations?view=hunt'] });
    expect(await screen.findByTestId('liquidation-hunt')).toBeInTheDocument();
    expect(screen.queryByTestId('liquidation-hunt-path-up')).not.toBeInTheDocument();
  });

  it('opens the hunt cascade path from the URL', async () => {
    renderWithProviders(<LiquidationsPage />, { routerEntries: ['/liquidations?view=hunt&panel=path'] });
    expect(await screen.findByTestId('liquidation-hunt-path-up')).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-hunt-path-step')).toHaveTextContent('125x');
  });
});
